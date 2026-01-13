package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/auth-platform/file-upload/internal/domain"
	"github.com/auth-platform/file-upload/internal/observability"
	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	userContextKey contextKey = "user_context"
)

// Middleware provides authentication middleware for Gin
type Middleware struct {
	handler *Handler
	logger  *observability.Logger
	metrics *observability.Metrics
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(handler *Handler, logger *observability.Logger, metrics *observability.Metrics) *Middleware {
	return &Middleware{
		handler: handler,
		logger:  logger,
		metrics: metrics,
	}
}

// Authenticate returns a Gin middleware that validates JWT tokens
func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.metrics.RecordAuthFailure("missing_token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    domain.ErrCodeMissingToken,
				"message": "authorization header is required",
			})
			return
		}

		// Validate token
		userCtx, err := m.handler.ValidateToken(c.Request.Context(), authHeader)
		if err != nil {
			statusCode := http.StatusUnauthorized
			errCode := domain.GetErrorCode(err)

			switch {
			case domain.IsAuthError(err):
				m.metrics.RecordAuthFailure(errCode)
			default:
				m.metrics.RecordAuthFailure("unknown")
			}

			c.AbortWithStatusJSON(statusCode, gin.H{
				"code":    errCode,
				"message": err.Error(),
			})
			return
		}

		// Store user context
		ctx := context.WithValue(c.Request.Context(), userContextKey, userCtx)
		ctx = observability.WithTenantID(ctx, userCtx.TenantID)
		ctx = observability.WithUserID(ctx, userCtx.UserID)
		c.Request = c.Request.WithContext(ctx)

		// Also store in Gin context for easy access
		c.Set("user_context", userCtx)
		c.Set("tenant_id", userCtx.TenantID)
		c.Set("user_id", userCtx.UserID)

		c.Next()
	}
}

// RequireRole returns a middleware that checks for a specific role
func (m *Middleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c.Request.Context())
		if userCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    domain.ErrCodeMissingToken,
				"message": "authentication required",
			})
			return
		}

		if !userCtx.HasRole(role) {
			m.metrics.RecordAuthFailure("insufficient_role")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    domain.ErrCodeAccessDenied,
				"message": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RequireTenantAccess returns a middleware that validates tenant access
func (m *Middleware) RequireTenantAccess(tenantIDParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c.Request.Context())
		if userCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    domain.ErrCodeMissingToken,
				"message": "authentication required",
			})
			return
		}

		// Get tenant ID from path parameter or query
		resourceTenantID := c.Param(tenantIDParam)
		if resourceTenantID == "" {
			resourceTenantID = c.Query(tenantIDParam)
		}

		// If no tenant ID in request, use user's tenant
		if resourceTenantID == "" {
			c.Next()
			return
		}

		// Validate access
		if err := m.handler.AuthorizeAccess(c.Request.Context(), userCtx, resourceTenantID); err != nil {
			m.metrics.RecordAuthFailure("tenant_mismatch")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    domain.ErrCodeAccessDenied,
				"message": "access denied to this resource",
			})
			return
		}

		c.Next()
	}
}

// GetUserContext retrieves user context from request context
func GetUserContext(ctx context.Context) *UserContext {
	if userCtx, ok := ctx.Value(userContextKey).(*UserContext); ok {
		return userCtx
	}
	return nil
}

// GetUserContextFromGin retrieves user context from Gin context
func GetUserContextFromGin(c *gin.Context) *UserContext {
	if userCtx, exists := c.Get("user_context"); exists {
		if uc, ok := userCtx.(*UserContext); ok {
			return uc
		}
	}
	return nil
}

// GetTenantID retrieves tenant ID from Gin context
func GetTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get("tenant_id"); exists {
		if tid, ok := tenantID.(string); ok {
			return tid
		}
	}
	return ""
}

// GetUserID retrieves user ID from Gin context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

// ExtractBearerToken extracts the token from Authorization header
func ExtractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	
	return strings.TrimSpace(parts[1])
}
