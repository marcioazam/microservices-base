// Package s3 provides AWS S3 storage implementation.
package s3

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/auth-platform/file-upload/internal/infrastructure/storage"
)

// Provider implements storage.Storage using AWS S3.
type Provider struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	pathBuilder   *storage.PathBuilder

	// Circuit breaker state
	circuitOpen   bool
	failures      int
	failThreshold int
	resetTimeout  time.Duration
	lastFailure   time.Time
	mu            sync.RWMutex
}

// Config holds S3 provider configuration.
type Config struct {
	Region        string
	Bucket        string
	Endpoint      string // For S3-compatible services
	FailThreshold int
	ResetTimeout  time.Duration
}

// NewProvider creates a new S3 storage provider.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *s3.Client
	if cfg.Endpoint != "" {
		// Use custom endpoint for S3-compatible services
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	return &Provider{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.Bucket,
		pathBuilder:   storage.NewPathBuilder(),
		failThreshold: cfg.FailThreshold,
		resetTimeout:  cfg.ResetTimeout,
	}, nil
}

// isCircuitOpen checks if circuit breaker is open.
func (p *Provider) isCircuitOpen() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.circuitOpen {
		return false
	}

	if time.Since(p.lastFailure) > p.resetTimeout {
		return false
	}

	return true
}

// recordFailure records a failure.
func (p *Provider) recordFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.failures++
	p.lastFailure = time.Now()
	if p.failures >= p.failThreshold {
		p.circuitOpen = true
	}
}

// recordSuccess records a success.
func (p *Provider) recordSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.failures = 0
	p.circuitOpen = false
}

// Upload uploads a file to S3.
func (p *Provider) Upload(ctx context.Context, req *storage.UploadRequest) (*storage.UploadResult, error) {
	if p.isCircuitOpen() {
		return nil, ErrStorageUnavailable
	}

	path := p.pathBuilder.BuildPath(req.TenantID, req.FileHash, req.Filename)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(path),
		Body:        req.Content,
		ContentType: aws.String(req.ContentType),
	}

	if len(req.Metadata) > 0 {
		input.Metadata = req.Metadata
	}

	output, err := p.client.PutObject(ctx, input)
	if err != nil {
		p.recordFailure()
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	p.recordSuccess()

	result := &storage.UploadResult{
		Path: path,
		URL:  fmt.Sprintf("s3://%s/%s", p.bucket, path),
	}

	if output.ETag != nil {
		result.ETag = *output.ETag
	}
	if output.VersionId != nil {
		result.VersionID = *output.VersionId
	}

	return result, nil
}

// Download downloads a file from S3.
func (p *Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if p.isCircuitOpen() {
		return nil, ErrStorageUnavailable
	}

	output, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		p.recordFailure()
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	p.recordSuccess()
	return output.Body, nil
}

// Delete deletes a file from S3.
func (p *Provider) Delete(ctx context.Context, path string) error {
	if p.isCircuitOpen() {
		return ErrStorageUnavailable
	}

	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		p.recordFailure()
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	p.recordSuccess()
	return nil
}

// GeneratePresignedUploadURL generates a presigned URL for upload.
func (p *Provider) GeneratePresignedUploadURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if p.isCircuitOpen() {
		return "", ErrStorageUnavailable
	}

	presignedReq, err := p.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		p.recordFailure()
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	p.recordSuccess()
	return presignedReq.URL, nil
}

// GeneratePresignedDownloadURL generates a presigned URL for download.
func (p *Provider) GeneratePresignedDownloadURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if p.isCircuitOpen() {
		return "", ErrStorageUnavailable
	}

	presignedReq, err := p.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		p.recordFailure()
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	p.recordSuccess()
	return presignedReq.URL, nil
}

// Exists checks if a file exists in S3.
func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
	if p.isCircuitOpen() {
		return false, ErrStorageUnavailable
	}

	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		// Check if it's a not found error
		p.recordSuccess() // Not found is not a failure
		return false, nil
	}

	p.recordSuccess()
	return true, nil
}

// GetMetadata retrieves object metadata from S3.
func (p *Provider) GetMetadata(ctx context.Context, path string) (*storage.ObjectMetadata, error) {
	if p.isCircuitOpen() {
		return nil, ErrStorageUnavailable
	}

	output, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		p.recordFailure()
		return nil, fmt.Errorf("failed to get metadata from S3: %w", err)
	}

	p.recordSuccess()

	metadata := &storage.ObjectMetadata{
		Path:     path,
		Metadata: output.Metadata,
	}

	if output.ContentLength != nil {
		metadata.Size = *output.ContentLength
	}
	if output.ContentType != nil {
		metadata.ContentType = *output.ContentType
	}
	if output.ETag != nil {
		metadata.ETag = *output.ETag
	}
	if output.LastModified != nil {
		metadata.LastModified = *output.LastModified
	}

	return metadata, nil
}

// GetPathBuilder returns the path builder.
func (p *Provider) GetPathBuilder() *storage.PathBuilder {
	return p.pathBuilder
}

// IsCircuitOpen returns true if circuit breaker is open.
func (p *Provider) IsCircuitOpen() bool {
	return p.isCircuitOpen()
}

// Errors
var (
	ErrStorageUnavailable = &StorageError{Code: "STORAGE_UNAVAILABLE", Message: "storage service is unavailable"}
)

// StorageError represents a storage operation error.
type StorageError struct {
	Code    string
	Message string
}

func (e *StorageError) Error() string {
	return e.Code + ": " + e.Message
}

// Is implements errors.Is.
func (e *StorageError) Is(target error) bool {
	t, ok := target.(*StorageError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}
