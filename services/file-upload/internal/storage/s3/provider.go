package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/auth-platform/file-upload/internal/config"
	"github.com/auth-platform/file-upload/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Provider implements cloud storage using AWS S3
type Provider struct {
	client          *s3.Client
	presignClient   *s3.PresignClient
	bucket          string
	region          string
	publicAccess    bool
	signedURLExpiry time.Duration
}

// StorageRequest contains upload parameters
type StorageRequest struct {
	TenantID    string
	FileHash    string
	Filename    string
	Content     io.Reader
	ContentType string
	Size        int64
	Metadata    map[string]string
}

// StorageResult contains upload result
type StorageResult struct {
	Path      string
	URL       string
	ETag      string
	VersionID string
}

// NewProvider creates a new S3 storage provider
func NewProvider(ctx context.Context, cfg config.StorageConfig) (*Provider, error) {
	var awsCfg aws.Config
	var err error

	// Configure AWS SDK
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	// Use custom credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	awsCfg, err = awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client options
	s3Opts := []func(*s3.Options){}

	// Custom endpoint for localstack/minio
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.UsePathStyle
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	presignClient := s3.NewPresignClient(client)

	return &Provider{
		client:          client,
		presignClient:   presignClient,
		bucket:          cfg.Bucket,
		region:          cfg.Region,
		publicAccess:    cfg.PublicAccess,
		signedURLExpiry: cfg.SignedURLExpiry,
	}, nil
}

// GeneratePath generates a storage path in the format:
// /{tenant_id}/{year}/{month}/{day}/{file_hash}/{filename}
func GeneratePath(tenantID, fileHash, filename string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s/%d/%02d/%02d/%s/%s",
		tenantID,
		now.Year(),
		now.Month(),
		now.Day(),
		fileHash,
		filename,
	)
}

// Upload uploads file to S3
func (p *Provider) Upload(ctx context.Context, req *StorageRequest) (*StorageResult, error) {
	path := GeneratePath(req.TenantID, req.FileHash, req.Filename)

	// Prepare metadata
	metadata := make(map[string]string)
	for k, v := range req.Metadata {
		metadata[k] = v
	}
	metadata["tenant-id"] = req.TenantID
	metadata["file-hash"] = req.FileHash

	input := &s3.PutObjectInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(path),
		Body:          req.Content,
		ContentType:   aws.String(req.ContentType),
		ContentLength: aws.Int64(req.Size),
		Metadata:      metadata,
	}

	result, err := p.client.PutObject(ctx, input)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeStorageError, "failed to upload file", err)
	}

	storageResult := &StorageResult{
		Path: path,
	}

	if result.ETag != nil {
		storageResult.ETag = *result.ETag
	}
	if result.VersionId != nil {
		storageResult.VersionID = *result.VersionId
	}

	// Generate URL
	if p.publicAccess {
		storageResult.URL = p.GeneratePublicURL(ctx, path)
	}

	return storageResult, nil
}

// Download retrieves file from storage
func (p *Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}

	result, err := p.client.GetObject(ctx, input)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeStorageError, "failed to download file", err)
	}

	return result.Body, nil
}

// Delete removes file from storage
func (p *Provider) Delete(ctx context.Context, path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}

	_, err := p.client.DeleteObject(ctx, input)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeStorageError, "failed to delete file", err)
	}

	return nil
}

// GenerateSignedURL creates time-limited access URL
func (p *Provider) GenerateSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = p.signedURLExpiry
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}

	presignResult, err := p.presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", domain.NewDomainError(domain.ErrCodeStorageError, "failed to generate signed URL", err)
	}

	return presignResult.URL, nil
}

// GeneratePublicURL creates permanent public URL
func (p *Provider) GeneratePublicURL(ctx context.Context, path string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", p.bucket, p.region, path)
}

// Exists checks if file exists
func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}

	_, err := p.client.HeadObject(ctx, input)
	if err != nil {
		// Check if it's a not found error
		return false, nil
	}

	return true, nil
}

// GetBucket returns the bucket name
func (p *Provider) GetBucket() string {
	return p.bucket
}

// GetRegion returns the region
func (p *Provider) GetRegion() string {
	return p.region
}
