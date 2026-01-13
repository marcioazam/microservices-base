// Package testutil provides test utilities and mocks for IAM Policy Service.
package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockCache implements a simple in-memory cache for testing.
type MockCache struct {
	mu    sync.RWMutex
	data  map[string][]byte
	ttls  map[string]time.Time
	calls []CacheCall
}

// CacheCall records a cache operation for verification.
type CacheCall struct {
	Method string
	Key    string
	Value  []byte
	TTL    time.Duration
}

// NewMockCache creates a new mock cache.
func NewMockCache() *MockCache {
	return &MockCache{
		data:  make(map[string][]byte),
		ttls:  make(map[string]time.Time),
		calls: make([]CacheCall, 0),
	}
}

// Get retrieves a value from the mock cache.
func (m *MockCache) Get(_ context.Context, key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls = append(m.calls, CacheCall{Method: "Get", Key: key})

	if expiry, ok := m.ttls[key]; ok && time.Now().After(expiry) {
		return nil, false
	}

	val, ok := m.data[key]
	return val, ok
}

// Set stores a value in the mock cache.
func (m *MockCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, CacheCall{Method: "Set", Key: key, Value: value, TTL: ttl})
	m.data[key] = value
	if ttl > 0 {
		m.ttls[key] = time.Now().Add(ttl)
	}
	return nil
}

// Delete removes a value from the mock cache.
func (m *MockCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, CacheCall{Method: "Delete", Key: key})
	delete(m.data, key)
	delete(m.ttls, key)
	return nil
}

// Invalidate clears all cache entries.
func (m *MockCache) Invalidate(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, CacheCall{Method: "Invalidate"})
	m.data = make(map[string][]byte)
	m.ttls = make(map[string]time.Time)
	return nil
}

// GetCalls returns all recorded cache calls.
func (m *MockCache) GetCalls() []CacheCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]CacheCall{}, m.calls...)
}

// Reset clears all data and recorded calls.
func (m *MockCache) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	m.ttls = make(map[string]time.Time)
	m.calls = make([]CacheCall, 0)
}

// MockLogger implements a simple logger for testing.
type MockLogger struct {
	mu      sync.RWMutex
	entries []LogEntry
}

// LogEntry represents a log entry for testing.
type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		entries: make([]LogEntry, 0),
	}
}

// Debug logs at debug level.
func (m *MockLogger) Debug(_ context.Context, msg string, fields map[string]interface{}) {
	m.log("DEBUG", msg, fields)
}

// Info logs at info level.
func (m *MockLogger) Info(_ context.Context, msg string, fields map[string]interface{}) {
	m.log("INFO", msg, fields)
}

// Warn logs at warn level.
func (m *MockLogger) Warn(_ context.Context, msg string, fields map[string]interface{}) {
	m.log("WARN", msg, fields)
}

// Error logs at error level.
func (m *MockLogger) Error(_ context.Context, msg string, fields map[string]interface{}) {
	m.log("ERROR", msg, fields)
}

func (m *MockLogger) log(level, msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, LogEntry{Level: level, Message: msg, Fields: fields})
}

// GetEntries returns all logged entries.
func (m *MockLogger) GetEntries() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]LogEntry{}, m.entries...)
}

// Reset clears all logged entries.
func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make([]LogEntry, 0)
}

// MockPolicyEvaluator implements a mock policy evaluator for testing.
type MockPolicyEvaluator struct {
	mu        sync.RWMutex
	cache     map[string]bool
	evalCount int
}

// PolicyStats holds policy evaluator statistics.
type PolicyStats struct {
	EvalCount int
	CacheSize int
}

// NewMockPolicyEvaluator creates a new mock policy evaluator.
func NewMockPolicyEvaluator() *MockPolicyEvaluator {
	return &MockPolicyEvaluator{
		cache: make(map[string]bool),
	}
}

// Evaluate evaluates a policy against input.
func (m *MockPolicyEvaluator) Evaluate(_ context.Context, input map[string]interface{}) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evalCount++

	// Generate deterministic key from input
	key := fmt.Sprintf("%v", input)

	// Check cache
	if result, ok := m.cache[key]; ok {
		return result, nil
	}

	// Deterministic evaluation based on role
	result := m.evaluateRole(input)
	m.cache[key] = result

	return result, nil
}

func (m *MockPolicyEvaluator) evaluateRole(input map[string]interface{}) bool {
	subject, ok := input["subject"].(map[string]interface{})
	if !ok {
		return false
	}

	attrs, ok := subject["attributes"].(map[string]interface{})
	if !ok {
		return false
	}

	role, _ := attrs["role"].(string)
	action, _ := input["action"].(string)

	// Simple RBAC logic for testing
	switch role {
	case "admin":
		return true
	case "editor":
		return action != "delete"
	case "viewer":
		return action == "read"
	default:
		return false
	}
}

// Reload simulates policy reload and clears cache.
func (m *MockPolicyEvaluator) Reload() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]bool)
}

// Stats returns evaluator statistics.
func (m *MockPolicyEvaluator) Stats() PolicyStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return PolicyStats{
		EvalCount: m.evalCount,
		CacheSize: len(m.cache),
	}
}


// AuthorizationRequest represents an authorization request for testing.
type AuthorizationRequest struct {
	SubjectID    string
	ResourceID   string
	ResourceType string
	Action       string
}

// AuthorizationResponse represents an authorization response for testing.
type AuthorizationResponse struct {
	Allowed     bool
	Reason      string
	EvaluatedAt time.Time
}

// MockAuthorizationService implements a mock authorization service for testing.
type MockAuthorizationService struct {
	mu    sync.RWMutex
	cache map[string]bool
}

// NewMockAuthorizationService creates a new mock authorization service.
func NewMockAuthorizationService() *MockAuthorizationService {
	return &MockAuthorizationService{
		cache: make(map[string]bool),
	}
}

// Authorize evaluates an authorization request.
func (m *MockAuthorizationService) Authorize(_ context.Context, req AuthorizationRequest) (*AuthorizationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s:%s", req.SubjectID, req.ResourceType, req.ResourceID, req.Action)

	if result, ok := m.cache[key]; ok {
		return &AuthorizationResponse{
			Allowed:     result,
			Reason:      "cached",
			EvaluatedAt: time.Now(),
		}, nil
	}

	// Simple deterministic logic
	allowed := req.Action == "read"
	m.cache[key] = allowed

	reason := "denied"
	if allowed {
		reason = "allowed"
	}

	return &AuthorizationResponse{
		Allowed:     allowed,
		Reason:      reason,
		EvaluatedAt: time.Now(),
	}, nil
}

// BatchAuthorize evaluates multiple authorization requests.
func (m *MockAuthorizationService) BatchAuthorize(ctx context.Context, requests []AuthorizationRequest) ([]AuthorizationResponse, error) {
	responses := make([]AuthorizationResponse, len(requests))
	for i, req := range requests {
		resp, err := m.Authorize(ctx, req)
		if err != nil {
			return nil, err
		}
		responses[i] = *resp
	}
	return responses, nil
}

// GetPermissions returns permissions for roles.
func (m *MockAuthorizationService) GetPermissions(_ context.Context, _ string, roles []string) ([]string, error) {
	permMap := map[string][]string{
		"admin":  {"read", "write", "delete", "create"},
		"editor": {"read", "write", "create"},
		"viewer": {"read"},
	}

	perms := make(map[string]bool)
	for _, role := range roles {
		if rolePerms, ok := permMap[role]; ok {
			for _, p := range rolePerms {
				perms[p] = true
			}
		}
	}

	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}
	return result, nil
}


// AuthorizeRequest for gRPC mock.
type AuthorizeRequest struct {
	SubjectID    string
	ResourceType string
	ResourceID   string
	Action       string
}

// AuthorizeResponseGRPC for gRPC mock.
type AuthorizeResponseGRPC struct {
	Allowed  bool
	PolicyID string
	Reason   string
}

// MockGRPCHandler implements a mock gRPC handler for testing.
type MockGRPCHandler struct{}

// NewMockGRPCHandler creates a new mock gRPC handler.
func NewMockGRPCHandler() *MockGRPCHandler {
	return &MockGRPCHandler{}
}

// Authorize handles authorization requests.
func (m *MockGRPCHandler) Authorize(_ context.Context, req *AuthorizeRequest) (*AuthorizeResponseGRPC, error) {
	if req.SubjectID == "" {
		return nil, fmt.Errorf("subject_id required")
	}
	if req.Action == "" {
		return nil, fmt.Errorf("action required")
	}
	if req.ResourceType == "" {
		return nil, fmt.Errorf("resource_type required")
	}

	return &AuthorizeResponseGRPC{
		Allowed:  req.Action == "read",
		PolicyID: "default",
		Reason:   "mock response",
	}, nil
}

// BatchAuthorize handles batch authorization requests.
func (m *MockGRPCHandler) BatchAuthorize(ctx context.Context, requests []*AuthorizeRequest) ([]*AuthorizeResponseGRPC, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no requests provided")
	}
	if len(requests) > 100 {
		return nil, fmt.Errorf("too many requests")
	}

	responses := make([]*AuthorizeResponseGRPC, len(requests))
	for i, req := range requests {
		resp, err := m.Authorize(ctx, req)
		if err != nil {
			responses[i] = &AuthorizeResponseGRPC{Allowed: false, Reason: err.Error()}
			continue
		}
		responses[i] = resp
	}
	return responses, nil
}

// GetUserPermissions returns permissions for a user.
func (m *MockGRPCHandler) GetUserPermissions(_ context.Context, userID string, roles []string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id required")
	}
	return []string{"read", "write"}, nil
}

// GetUserRoles returns roles for a user.
func (m *MockGRPCHandler) GetUserRoles(_ context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id required")
	}
	return []string{"viewer"}, nil
}

// ReloadPolicies triggers policy reload.
func (m *MockGRPCHandler) ReloadPolicies(_ context.Context) (bool, error) {
	return true, nil
}


// CAEPEvent represents a CAEP event for testing.
type CAEPEvent struct {
	EventType      string
	Subject        CAEPSubject
	EventTimestamp int64
	Extra          map[string]interface{}
}

// CAEPSubject represents a CAEP subject for testing.
type CAEPSubject struct {
	Format string
	Iss    string
	Sub    string
}

// NewMockCAEPEvent creates a mock CAEP event.
func NewMockCAEPEvent(eventType, userID string) *CAEPEvent {
	return &CAEPEvent{
		EventType: eventType,
		Subject: CAEPSubject{
			Format: "iss_sub",
			Iss:    "https://auth.example.com",
			Sub:    userID,
		},
		EventTimestamp: time.Now().Unix(),
		Extra:          make(map[string]interface{}),
	}
}

// NewMockCAEPEventWithIssuer creates a mock CAEP event with custom issuer.
func NewMockCAEPEventWithIssuer(eventType, userID, issuer string) *CAEPEvent {
	return &CAEPEvent{
		EventType: eventType,
		Subject: CAEPSubject{
			Format: "iss_sub",
			Iss:    issuer,
			Sub:    userID,
		},
		EventTimestamp: time.Now().Unix(),
		Extra:          make(map[string]interface{}),
	}
}

// NewMockAssuranceLevelChangeEvent creates a mock assurance level change event.
func NewMockAssuranceLevelChangeEvent(userID, previousLevel, currentLevel string) *CAEPEvent {
	event := NewMockCAEPEvent("assurance-level-change", userID)
	event.Extra["previous_level"] = previousLevel
	event.Extra["current_level"] = currentLevel
	return event
}

// NewMockTokenClaimsChangeEvent creates a mock token claims change event.
func NewMockTokenClaimsChangeEvent(userID string, changedClaims []string) *CAEPEvent {
	event := NewMockCAEPEvent("token-claims-change", userID)
	event.Extra["changed_claims"] = changedClaims
	return event
}


type traceContextKey struct{}
type spanContextKey struct{}

// ContextWithTrace creates a context with trace information.
func ContextWithTrace(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, traceContextKey{}, traceID)
	ctx = context.WithValue(ctx, spanContextKey{}, spanID)
	return ctx
}

// ExtractTraceContext extracts trace context from context.
func ExtractTraceContext(ctx context.Context) (traceID, spanID string) {
	if id, ok := ctx.Value(traceContextKey{}).(string); ok {
		traceID = id
	}
	if id, ok := ctx.Value(spanContextKey{}).(string); ok {
		spanID = id
	}
	return
}


// MockServiceError represents a service error for testing.
type MockServiceError struct {
	Code          string
	Message       string
	CorrelationID string
}

// Error implements the error interface.
func (e *MockServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewMockServiceError creates a mock service error.
func NewMockServiceError(code, message, correlationID string) *MockServiceError {
	return &MockServiceError{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
	}
}

// MapToGRPCCode maps error code to gRPC code name.
func MapToGRPCCode(code string) string {
	switch code {
	case "INVALID_INPUT":
		return "InvalidArgument"
	case "UNAUTHORIZED":
		return "Unauthenticated"
	case "FORBIDDEN":
		return "PermissionDenied"
	case "NOT_FOUND":
		return "NotFound"
	case "INTERNAL":
		return "Internal"
	case "UNAVAILABLE":
		return "Unavailable"
	case "RATE_LIMITED":
		return "ResourceExhausted"
	default:
		return "Unknown"
	}
}
