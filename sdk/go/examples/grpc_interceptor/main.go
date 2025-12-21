// Example: gRPC interceptor with Auth Platform SDK
package main

import (
	"context"
	"log"
	"net"

	authplatform "github.com/auth-platform/sdk-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// ExampleService implements a simple gRPC service
type ExampleService struct {
	UnimplementedExampleServiceServer
}

func (s *ExampleService) GetData(ctx context.Context, req *GetDataRequest) (*GetDataResponse, error) {
	// Get claims from context
	claims, ok := authplatform.GetClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no claims in context")
	}

	return &GetDataResponse{
		UserId: claims.Subject,
		Data:   []string{"item1", "item2"},
	}, nil
}

func main() {
	// Create auth client
	client, err := authplatform.New(authplatform.Config{
		BaseURL:  "https://auth.example.com",
		ClientID: "your-client-id",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create gRPC server with interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(client.UnaryServerInterceptor(
			authplatform.WithGRPCSkipMethods(
				"/grpc.health.v1.Health/Check",
				"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
			),
		)),
		grpc.StreamInterceptor(client.StreamServerInterceptor()),
	)

	// Register services
	RegisterExampleServiceServer(server, &ExampleService{})
	reflection.Register(server)

	// Start server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("gRPC server starting on :50051")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Placeholder types - in real usage, these would be generated from proto
type UnimplementedExampleServiceServer struct{}
type GetDataRequest struct{}
type GetDataResponse struct {
	UserId string
	Data   []string
}

func RegisterExampleServiceServer(s *grpc.Server, srv *ExampleService) {
	// Registration would be generated from proto
}
