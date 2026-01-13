// Package handlers provides gRPC handlers for IAM Policy Service.
package handlers

import (
	"context"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/auth-platform/iam-policy-service/internal/logging"
	"github.com/auth-platform/iam-policy-service/internal/service"
	pb "github.com/auth-platform/iam-policy-service/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IAMPolicyService implements the gRPC IAM Policy service.
type IAMPolicyService struct {
	pb.UnimplementedIAMPolicyServiceServer
	authService *service.AuthorizationService
	config      *config.Config
	logger      *logging.Logger
}

// IAMPolicyServiceConfig holds configuration for the gRPC handler.
type IAMPolicyServiceConfig struct {
	AuthService *service.AuthorizationService
	Config      *config.Config
	Logger      *logging.Logger
}

// NewIAMPolicyService creates a new IAM Policy gRPC service.
func NewIAMPolicyService(cfg IAMPolicyServiceConfig) *IAMPolicyService {
	return &IAMPolicyService{
		authService: cfg.AuthService,
		config:      cfg.Config,
		logger:      cfg.Logger,
	}
}

// Authorize handles authorization requests.
func (s *IAMPolicyService) Authorize(ctx context.Context, req *pb.AuthorizeRequest) (*pb.AuthorizeResponse, error) {
	if err := s.validateAuthorizeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	authReq := service.AuthorizationRequest{
		SubjectID:    req.SubjectId,
		SubjectType:  req.SubjectType,
		ResourceID:   req.ResourceId,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Context:      convertAttributes(req.Environment),
	}

	resp, err := s.authService.Authorize(ctx, authReq)
	if err != nil {
		s.logError(ctx, "authorize", err)
		return nil, status.Error(codes.Internal, "authorization failed")
	}

	return &pb.AuthorizeResponse{
		Allowed:  resp.Allowed,
		PolicyId: resp.PolicyName,
		Reason:   resp.Reason,
	}, nil
}

// BatchAuthorize handles batch authorization requests.
func (s *IAMPolicyService) BatchAuthorize(ctx context.Context, req *pb.BatchAuthorizeRequest) (*pb.BatchAuthorizeResponse, error) {
	if len(req.Requests) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no requests provided")
	}

	if len(req.Requests) > 100 {
		return nil, status.Error(codes.InvalidArgument, "too many requests (max 100)")
	}

	responses := make([]*pb.AuthorizeResponse, len(req.Requests))

	for i, r := range req.Requests {
		resp, err := s.Authorize(ctx, r)
		if err != nil {
			// Continue with other requests, mark this one as denied
			responses[i] = &pb.AuthorizeResponse{
				Allowed: false,
				Reason:  "evaluation error",
			}
			continue
		}
		responses[i] = resp
	}

	return &pb.BatchAuthorizeResponse{Responses: responses}, nil
}

// GetUserPermissions returns permissions for a user.
func (s *IAMPolicyService) GetUserPermissions(ctx context.Context, req *pb.GetPermissionsRequest) (*pb.PermissionsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
	}

	permissions, err := s.authService.GetPermissions(ctx, req.UserId, req.Roles)
	if err != nil {
		s.logError(ctx, "get_permissions", err)
		return nil, status.Error(codes.Internal, "failed to get permissions")
	}

	pbPerms := make([]*pb.Permission, len(permissions))
	for i, p := range permissions {
		pbPerms[i] = &pb.Permission{Name: p}
	}

	return &pb.PermissionsResponse{Permissions: pbPerms}, nil
}

// GetUserRoles returns roles for a user.
func (s *IAMPolicyService) GetUserRoles(ctx context.Context, req *pb.GetRolesRequest) (*pb.RolesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
	}

	roles, err := s.authService.GetRoles(ctx, req.UserId)
	if err != nil {
		s.logError(ctx, "get_roles", err)
		return nil, status.Error(codes.Internal, "failed to get roles")
	}

	pbRoles := make([]*pb.Role, len(roles))
	for i, r := range roles {
		pbRoles[i] = &pb.Role{Id: r}
	}

	return &pb.RolesResponse{Roles: pbRoles}, nil
}

// ReloadPolicies triggers policy reload.
func (s *IAMPolicyService) ReloadPolicies(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error) {
	start := time.Now()

	if err := s.authService.ReloadPolicies(ctx); err != nil {
		s.logError(ctx, "reload_policies", err)
		return &pb.ReloadResponse{
			Success:      false,
			ErrorMessage: "failed to reload policies",
		}, nil
	}

	if s.logger != nil {
		s.logger.Info(ctx, "policies reloaded",
			logging.Duration("duration", time.Since(start)))
	}

	return &pb.ReloadResponse{Success: true}, nil
}

func (s *IAMPolicyService) validateAuthorizeRequest(req *pb.AuthorizeRequest) error {
	if req.SubjectId == "" {
		return status.Error(codes.InvalidArgument, "subject_id required")
	}
	if req.Action == "" {
		return status.Error(codes.InvalidArgument, "action required")
	}
	if req.ResourceType == "" {
		return status.Error(codes.InvalidArgument, "resource_type required")
	}
	return nil
}

func (s *IAMPolicyService) logError(ctx context.Context, operation string, err error) {
	if s.logger != nil {
		s.logger.Error(ctx, "grpc operation failed",
			logging.String("operation", operation),
			logging.Error(err))
	}
}

func convertAttributes(attrs map[string]string) map[string]interface{} {
	if attrs == nil {
		return nil
	}
	result := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		result[k] = v
	}
	return result
}
