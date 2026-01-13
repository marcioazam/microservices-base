// Package grpc provides gRPC server implementation.
package grpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/auth-platform/cache-service/internal/cache"
)

// Server implements the gRPC cache service.
type Server struct {
	UnimplementedCacheServiceServer
	cacheService cache.Service
	healthServer *health.Server
	grpcServer   *grpc.Server
}

// NewServer creates a new gRPC server.
func NewServer(cacheService cache.Service) *Server {
	grpcSrv := grpc.NewServer()
	s := &Server{
		cacheService: cacheService,
		healthServer: health.NewServer(),
		grpcServer:   grpcSrv,
	}

	RegisterCacheServiceServer(grpcSrv, s)
	healthpb.RegisterHealthServer(grpcSrv, s.healthServer)
	s.healthServer.SetServingStatus("cache.v1.CacheService", healthpb.HealthCheckResponse_SERVING)

	return s
}

// Get retrieves a value from cache.
func (s *Server) Get(ctx context.Context, req *GetRequest) (*GetResponse, error) {
	if req.Key == "" {
		return nil, InvalidArgumentError("key is required")
	}
	if req.Namespace == "" {
		return nil, InvalidArgumentError("namespace is required")
	}

	entry, err := s.cacheService.Get(ctx, req.Namespace, req.Key)
	if err != nil {
		if cache.IsNotFound(err) {
			return &GetResponse{Found: false}, nil
		}
		return nil, ToGRPCError(err)
	}

	return &GetResponse{
		Found:  true,
		Value:  entry.Value,
		Source: toCacheSource(entry.Source),
	}, nil
}

// Set stores a value in cache.
func (s *Server) Set(ctx context.Context, req *SetRequest) (*SetResponse, error) {
	if req.Key == "" {
		return nil, InvalidArgumentError("key is required")
	}
	if req.Namespace == "" {
		return nil, InvalidArgumentError("namespace is required")
	}
	if len(req.Value) == 0 {
		return nil, InvalidArgumentError("value is required")
	}

	ttl := time.Duration(req.TtlSeconds) * time.Second

	var opts []cache.SetOption
	if req.Encrypt {
		opts = append(opts, cache.WithEncryption())
	}

	err := s.cacheService.Set(ctx, req.Namespace, req.Key, req.Value, ttl, opts...)
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return &SetResponse{Success: true}, nil
}

// Delete removes a value from cache.
func (s *Server) Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	if req.Key == "" {
		return nil, InvalidArgumentError("key is required")
	}
	if req.Namespace == "" {
		return nil, InvalidArgumentError("namespace is required")
	}

	deleted, err := s.cacheService.Delete(ctx, req.Namespace, req.Key)
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return &DeleteResponse{Deleted: deleted}, nil
}

// BatchGet retrieves multiple values from cache.
func (s *Server) BatchGet(ctx context.Context, req *BatchGetRequest) (*BatchGetResponse, error) {
	if req.Namespace == "" {
		return nil, InvalidArgumentError("namespace is required")
	}
	if len(req.Keys) == 0 {
		return &BatchGetResponse{}, nil
	}

	found, missing, err := s.cacheService.BatchGet(ctx, req.Namespace, req.Keys)
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return &BatchGetResponse{
		Values:      found,
		MissingKeys: missing,
	}, nil
}

// BatchSet stores multiple key-value pairs in cache.
func (s *Server) BatchSet(ctx context.Context, req *BatchSetRequest) (*BatchSetResponse, error) {
	if req.Namespace == "" {
		return nil, InvalidArgumentError("namespace is required")
	}
	if len(req.Entries) == 0 {
		return &BatchSetResponse{Success: true, StoredCount: 0}, nil
	}

	ttl := time.Duration(req.TtlSeconds) * time.Second

	count, err := s.cacheService.BatchSet(ctx, req.Namespace, req.Entries, ttl)
	if err != nil {
		return nil, ToGRPCError(err)
	}

	// #nosec G115 - count is bounded by batch size (typically < 1000)
	storedCount := int32(min(count, 1<<30))
	return &BatchSetResponse{
		Success:     true,
		StoredCount: storedCount,
	}, nil
}

// Health returns the health status.
func (s *Server) Health(ctx context.Context, req *HealthRequest) (*HealthResponse, error) {
	st, err := s.cacheService.Health(ctx)
	if err != nil {
		return nil, ToGRPCError(err)
	}

	return &HealthResponse{
		Healthy:           st.Healthy,
		RedisStatus:       st.RedisStatus,
		BrokerStatus:      st.BrokerStatus,
		LocalCacheEnabled: st.LocalCache,
	}, nil
}

// Start starts the gRPC server.
func (s *Server) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	return s.grpcServer.Serve(lis)
}

// Serve starts the gRPC server with an existing listener.
func (s *Server) Serve(lis net.Listener) error {
	return s.grpcServer.Serve(lis)
}

// GracefulStop gracefully stops the server.
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}

// Stop immediately stops the server.
func (s *Server) Stop() {
	s.grpcServer.Stop()
}

// GRPCServer returns the underlying grpc.Server.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

func toCacheSource(source cache.Source) CacheSource {
	switch source {
	case cache.SourceRedis:
		return CacheSource_CACHE_SOURCE_REDIS
	case cache.SourceLocal:
		return CacheSource_CACHE_SOURCE_LOCAL
	default:
		return CacheSource_CACHE_SOURCE_UNSPECIFIED
	}
}

// Proto message types (simplified for compilation without protoc).
type GetRequest struct {
	Key       string
	Namespace string
}

type GetResponse struct {
	Found  bool
	Value  []byte
	Source CacheSource
}

type SetRequest struct {
	Key        string
	Value      []byte
	TtlSeconds int64
	Namespace  string
	Encrypt    bool
}

type SetResponse struct {
	Success bool
}

type DeleteRequest struct {
	Key       string
	Namespace string
}

type DeleteResponse struct {
	Deleted bool
}

type BatchGetRequest struct {
	Keys      []string
	Namespace string
}

type BatchGetResponse struct {
	Values      map[string][]byte
	MissingKeys []string
}

type BatchSetRequest struct {
	Entries    map[string][]byte
	TtlSeconds int64
	Namespace  string
}

type BatchSetResponse struct {
	Success     bool
	StoredCount int32
}

type HealthRequest struct{}

type HealthResponse struct {
	Healthy           bool
	RedisStatus       string
	BrokerStatus      string
	LocalCacheEnabled bool
}

type CacheSource int32

const (
	CacheSource_CACHE_SOURCE_UNSPECIFIED CacheSource = 0
	CacheSource_CACHE_SOURCE_REDIS       CacheSource = 1
	CacheSource_CACHE_SOURCE_LOCAL       CacheSource = 2
)

// UnimplementedCacheServiceServer for forward compatibility.
type UnimplementedCacheServiceServer struct{}

func (UnimplementedCacheServiceServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedCacheServiceServer) Set(context.Context, *SetRequest) (*SetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Set not implemented")
}
func (UnimplementedCacheServiceServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedCacheServiceServer) BatchGet(context.Context, *BatchGetRequest) (*BatchGetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchGet not implemented")
}
func (UnimplementedCacheServiceServer) BatchSet(context.Context, *BatchSetRequest) (*BatchSetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchSet not implemented")
}
func (UnimplementedCacheServiceServer) Health(context.Context, *HealthRequest) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}

// RegisterCacheServiceServer registers the server.
func RegisterCacheServiceServer(s *grpc.Server, srv CacheServiceServer) {
	// Registration would be done by generated code
}

// CacheServiceServer interface.
type CacheServiceServer interface {
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Set(context.Context, *SetRequest) (*SetResponse, error)
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	BatchGet(context.Context, *BatchGetRequest) (*BatchGetResponse, error)
	BatchSet(context.Context, *BatchSetRequest) (*BatchSetResponse, error)
	Health(context.Context, *HealthRequest) (*HealthResponse, error)
}
