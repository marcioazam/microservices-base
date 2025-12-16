package handlers

import (
	"context"
	"log"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/auth-platform/iam-policy-service/internal/policy"
	"github.com/auth-platform/iam-policy-service/internal/rbac"
	pb "github.com/auth-platform/iam-policy-service/proto"
)

type IAMPolicyService struct {
	pb.UnimplementedIAMPolicyServiceServer
	engine    *policy.Engine
	hierarchy *rbac.RoleHierarchy
	config    *config.Config
}

func NewIAMPolicyService(engine *policy.Engine, cfg *config.Config) *IAMPolicyService {
	return &IAMPolicyService{
		engine:    engine,
		hierarchy: rbac.NewRoleHierarchy(),
		config:    cfg,
	}
}

func (s *IAMPolicyService) Authorize(ctx context.Context, req *pb.AuthorizeRequest) (*pb.AuthorizeResponse, error) {
	start := time.Now()

	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"id":         req.SubjectId,
			"attributes": req.SubjectAttributes,
		},
		"resource": map[string]interface{}{
			"type":       req.ResourceType,
			"id":         req.ResourceId,
			"attributes": req.ResourceAttributes,
		},
		"action":      req.Action,
		"environment": req.Environment,
	}

	allowed, policyID, matchedRules, err := s.engine.Evaluate(ctx, input)
	if err != nil {
		return nil, err
	}

	duration := time.Since(start)

	if s.config.AuditLogEnabled {
		log.Printf("Authorization decision: subject=%s resource=%s/%s action=%s allowed=%v duration=%v",
			req.SubjectId, req.ResourceType, req.ResourceId, req.Action, allowed, duration)
	}

	reason := "Denied by default"
	if allowed {
		reason = "Allowed by policy"
	}

	return &pb.AuthorizeResponse{
		Allowed:      allowed,
		PolicyId:     policyID,
		Reason:       reason,
		MatchedRules: matchedRules,
	}, nil
}

func (s *IAMPolicyService) BatchAuthorize(ctx context.Context, req *pb.BatchAuthorizeRequest) (*pb.BatchAuthorizeResponse, error) {
	responses := make([]*pb.AuthorizeResponse, len(req.Requests))

	for i, r := range req.Requests {
		resp, err := s.Authorize(ctx, r)
		if err != nil {
			return nil, err
		}
		responses[i] = resp
	}

	return &pb.BatchAuthorizeResponse{
		Responses: responses,
	}, nil
}

func (s *IAMPolicyService) GetUserPermissions(ctx context.Context, req *pb.GetPermissionsRequest) (*pb.PermissionsResponse, error) {
	// In production, this would query the database for user roles
	// and resolve permissions through the hierarchy
	return &pb.PermissionsResponse{
		Permissions: []*pb.Permission{},
	}, nil
}

func (s *IAMPolicyService) GetUserRoles(ctx context.Context, req *pb.GetRolesRequest) (*pb.RolesResponse, error) {
	// In production, this would query the database
	return &pb.RolesResponse{
		Roles: []*pb.Role{},
	}, nil
}

func (s *IAMPolicyService) ReloadPolicies(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error) {
	// Trigger policy reload
	count := s.engine.GetPolicyCount()

	return &pb.ReloadResponse{
		Success:        true,
		PoliciesLoaded: int32(count),
		ErrorMessage:   "",
	}, nil
}
