# Design Patterns (Continued)

Additional state-of-the-art reusable patterns for the Go library modernization.

## Pattern 7: Generic Cursor-Based Pagination

Efficient pagination for large datasets with stable ordering.

```go
// Cursor encodes pagination position.
type Cursor string

// Page represents a paginated result.
type Page[T any] struct {
    Items      []T
    NextCursor Option[Cursor]
    PrevCursor Option[Cursor]
    HasMore    bool
    TotalCount Option[int64]
}

// PageRequest specifies pagination parameters.
type PageRequest struct {
    Cursor    Option[Cursor]
    Limit     int
    Direction Direction
}

type Direction int

const (
    Forward Direction = iota
    Backward
)

// Paginator provides cursor-based pagination.
type Paginator[T any, K comparable] struct {
    keyFn    func(T) K
    encodeFn func(K) Cursor
    decodeFn func(Cursor) (K, error)
}

// NewPaginator creates a paginator for type T.
func NewPaginator[T any, K comparable](
    keyFn func(T) K,
    encodeFn func(K) Cursor,
    decodeFn func(Cursor) (K, error),
) *Paginator[T, K] {
    return &Paginator[T, K]{keyFn, encodeFn, decodeFn}
}

// Paginate applies pagination to items.
func (p *Paginator[T, K]) Paginate(items []T, req PageRequest) Page[T] {
    if len(items) == 0 {
        return Page[T]{Items: []T{}, HasMore: false}
    }
    
    hasMore := len(items) > req.Limit
    if hasMore {
        items = items[:req.Limit]
    }
    
    var nextCursor, prevCursor Option[Cursor]
    if hasMore && len(items) > 0 {
        lastKey := p.keyFn(items[len(items)-1])
        nextCursor = Some(p.encodeFn(lastKey))
    }
    if req.Cursor.IsSome() && len(items) > 0 {
        firstKey := p.keyFn(items[0])
        prevCursor = Some(p.encodeFn(firstKey))
    }
    
    return Page[T]{
        Items:      items,
        NextCursor: nextCursor,
        PrevCursor: prevCursor,
        HasMore:    hasMore,
    }
}
```

**Validates: Requirements 12.2**

## Pattern 8: Generic Health Check System

Kubernetes-compatible health probes with dependency checks.

```go
// HealthStatus represents component health.
type HealthStatus string

const (
    StatusHealthy   HealthStatus = "healthy"
    StatusDegraded  HealthStatus = "degraded"
    StatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck defines a health check function.
type HealthCheck func(ctx context.Context) HealthResult

// HealthResult contains check outcome.
type HealthResult struct {
    Status   HealthStatus
    Message  string
    Duration time.Duration
    Details  map[string]any
}

// HealthChecker manages multiple health checks.
type HealthChecker struct {
    checks   map[string]HealthCheck
    timeout  time.Duration
    mu       sync.RWMutex
}

// NewHealthChecker creates a health checker.
func NewHealthChecker(timeout time.Duration) *HealthChecker {
    return &HealthChecker{
        checks:  make(map[string]HealthCheck),
        timeout: timeout,
    }
}

// Register adds a named health check.
func (h *HealthChecker) Register(name string, check HealthCheck) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.checks[name] = check
}

// CheckAll runs all health checks concurrently.
func (h *HealthChecker) CheckAll(ctx context.Context) map[string]HealthResult {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    results := make(map[string]HealthResult)
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    for name, check := range h.checks {
        wg.Add(1)
        go func(n string, c HealthCheck) {
            defer wg.Done()
            
            checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
            defer cancel()
            
            start := time.Now()
            result := c(checkCtx)
            result.Duration = time.Since(start)
            
            mu.Lock()
            results[n] = result
            mu.Unlock()
        }(name, check)
    }
    
    wg.Wait()
    return results
}

// Liveness returns liveness probe result.
func (h *HealthChecker) Liveness(ctx context.Context) HealthResult {
    return HealthResult{Status: StatusHealthy, Message: "alive"}
}

// Readiness returns readiness probe result.
func (h *HealthChecker) Readiness(ctx context.Context) HealthResult {
    results := h.CheckAll(ctx)
    for _, r := range results {
        if r.Status == StatusUnhealthy {
            return HealthResult{Status: StatusUnhealthy, Message: "not ready"}
        }
    }
    return HealthResult{Status: StatusHealthy, Message: "ready"}
}
```

**Validates: Requirements 13.1**

## Pattern 9: Generic Outbox Pattern

Transactional messaging for event-driven systems.

```go
// OutboxMessage represents a pending message.
type OutboxMessage[T any] struct {
    ID          string
    AggregateID string
    EventType   string
    Payload     T
    CreatedAt   time.Time
    ProcessedAt Option[time.Time]
    Retries     int
}

// Outbox manages transactional message publishing.
type Outbox[T any] struct {
    store     OutboxStore[T]
    publisher Publisher[T]
    batchSize int
    interval  time.Duration
}

// OutboxStore persists outbox messages.
type OutboxStore[T any] interface {
    Save(ctx context.Context, msg OutboxMessage[T]) error
    FindPending(ctx context.Context, limit int) ([]OutboxMessage[T], error)
    MarkProcessed(ctx context.Context, id string) error
    IncrementRetry(ctx context.Context, id string) error
}

// Publisher sends messages to external systems.
type Publisher[T any] interface {
    Publish(ctx context.Context, msg OutboxMessage[T]) error
}

// NewOutbox creates an outbox processor.
func NewOutbox[T any](store OutboxStore[T], pub Publisher[T], opts ...OutboxOption) *Outbox[T] {
    o := &Outbox[T]{
        store:     store,
        publisher: pub,
        batchSize: 100,
        interval:  time.Second,
    }
    for _, opt := range opts {
        opt(o)
    }
    return o
}

// Process polls and publishes pending messages.
func (o *Outbox[T]) Process(ctx context.Context) error {
    messages, err := o.store.FindPending(ctx, o.batchSize)
    if err != nil {
        return err
    }
    
    for _, msg := range messages {
        if err := o.publisher.Publish(ctx, msg); err != nil {
            _ = o.store.IncrementRetry(ctx, msg.ID)
            continue
        }
        if err := o.store.MarkProcessed(ctx, msg.ID); err != nil {
            return err
        }
    }
    return nil
}

// Start begins background processing.
func (o *Outbox[T]) Start(ctx context.Context) {
    ticker := time.NewTicker(o.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            _ = o.Process(ctx)
        }
    }
}
```

**Validates: Requirements 15.2, 15.5**

## Pattern 10: Generic Builder with Validation

Fluent builder pattern with compile-time safety.

```go
// Builder constructs objects with validation.
type Builder[T any] struct {
    build  func() T
    errors []error
}

// NewBuilder creates a builder for type T.
func NewBuilder[T any](initial T) *Builder[T] {
    return &Builder[T]{
        build: func() T { return initial },
    }
}

// With applies a transformation.
func (b *Builder[T]) With(fn func(*T)) *Builder[T] {
    prev := b.build
    b.build = func() T {
        t := prev()
        fn(&t)
        return t
    }
    return b
}

// Validate adds a validation rule.
func (b *Builder[T]) Validate(check func(T) error) *Builder[T] {
    prev := b.build
    b.build = func() T {
        t := prev()
        if err := check(t); err != nil {
            b.errors = append(b.errors, err)
        }
        return t
    }
    return b
}

// Build returns the constructed object or errors.
func (b *Builder[T]) Build() Result[T] {
    result := b.build()
    if len(b.errors) > 0 {
        return Err[T](NewValidationError(b.errors))
    }
    return Ok(result)
}

// Example usage:
// config := NewBuilder(Config{}).
//     With(func(c *Config) { c.Timeout = 30 * time.Second }).
//     With(func(c *Config) { c.MaxRetries = 3 }).
//     Validate(func(c Config) error {
//         if c.Timeout <= 0 { return errors.New("timeout required") }
//         return nil
//     }).
//     Build()
```

**Validates: Requirements 5.2, 5.4**

## Pattern 11: Generic Circuit Breaker Composition

Composable resilience with multiple strategies.

```go
// Policy represents a resilience policy.
type Policy[T any] interface {
    Execute(ctx context.Context, fn func() (T, error)) (T, error)
}

// CompositePolicy chains multiple policies.
type CompositePolicy[T any] struct {
    policies []Policy[T]
}

// Compose creates a composite policy (outer to inner).
func Compose[T any](policies ...Policy[T]) Policy[T] {
    return &CompositePolicy[T]{policies: policies}
}

func (c *CompositePolicy[T]) Execute(ctx context.Context, fn func() (T, error)) (T, error) {
    if len(c.policies) == 0 {
        return fn()
    }
    
    wrapped := fn
    for i := len(c.policies) - 1; i >= 0; i-- {
        policy := c.policies[i]
        prev := wrapped
        wrapped = func() (T, error) {
            return policy.Execute(ctx, prev)
        }
    }
    return wrapped()
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker[T any] struct {
    name          string
    threshold     int
    resetTimeout  time.Duration
    state         atomic.Value // State
    failures      atomic.Int32
    lastFailure   atomic.Value // time.Time
}

func (cb *CircuitBreaker[T]) Execute(ctx context.Context, fn func() (T, error)) (T, error) {
    state := cb.state.Load().(State)
    
    switch state {
    case StateOpen:
        if time.Since(cb.lastFailure.Load().(time.Time)) > cb.resetTimeout {
            cb.state.Store(StateHalfOpen)
        } else {
            var zero T
            return zero, NewCircuitOpenError(cb.name, cb.resetTimeout)
        }
    }
    
    result, err := fn()
    if err != nil {
        cb.recordFailure()
        return result, err
    }
    
    cb.recordSuccess()
    return result, nil
}

// Usage: Compose retry inside circuit breaker
// policy := Compose[Response](
//     NewCircuitBreaker[Response]("api", 5, 30*time.Second),
//     NewRetryPolicy[Response](3, time.Second),
// )
// result, err := policy.Execute(ctx, callAPI)
```

**Validates: Requirements 4.2, 5.1**

## Pattern 12: Generic Saga Orchestrator

Distributed transaction coordination with compensation.

```go
// SagaStep represents a step in a saga.
type SagaStep[Ctx any] struct {
    Name       string
    Execute    func(ctx context.Context, data *Ctx) error
    Compensate func(ctx context.Context, data *Ctx) error
}

// Saga orchestrates distributed transactions.
type Saga[Ctx any] struct {
    steps     []SagaStep[Ctx]
    completed []int
}

// NewSaga creates a saga with steps.
func NewSaga[Ctx any](steps ...SagaStep[Ctx]) *Saga[Ctx] {
    return &Saga[Ctx]{steps: steps}
}

// Run executes the saga with automatic compensation on failure.
func (s *Saga[Ctx]) Run(ctx context.Context, data *Ctx) error {
    s.completed = nil
    
    for i, step := range s.steps {
        if err := step.Execute(ctx, data); err != nil {
            // Compensate in reverse order
            for j := len(s.completed) - 1; j >= 0; j-- {
                idx := s.completed[j]
                if compErr := s.steps[idx].Compensate(ctx, data); compErr != nil {
                    return NewSagaCompensationError(step.Name, err, compErr)
                }
            }
            return NewSagaExecutionError(step.Name, err)
        }
        s.completed = append(s.completed, i)
    }
    return nil
}

// Example:
// saga := NewSaga[OrderContext](
//     SagaStep[OrderContext]{
//         Name:       "reserve_inventory",
//         Execute:    reserveInventory,
//         Compensate: releaseInventory,
//     },
//     SagaStep[OrderContext]{
//         Name:       "charge_payment",
//         Execute:    chargePayment,
//         Compensate: refundPayment,
//     },
// )
```

**Validates: Requirements 15.5**

## Pattern 13: Observability Integration

Unified tracing, metrics, and logging.

```go
// Span represents a trace span.
type Span interface {
    SetAttribute(key string, value any)
    RecordError(err error)
    End()
}

// Tracer creates spans.
type Tracer interface {
    Start(ctx context.Context, name string) (context.Context, Span)
}

// Metrics records measurements.
type Metrics interface {
    Counter(name string) Counter
    Histogram(name string) Histogram
    Gauge(name string) Gauge
}

// Instrumented wraps operations with observability.
func Instrumented[T any](
    tracer Tracer,
    metrics Metrics,
    name string,
) func(func(context.Context) (T, error)) func(context.Context) (T, error) {
    counter := metrics.Counter(name + "_total")
    histogram := metrics.Histogram(name + "_duration_seconds")
    
    return func(fn func(context.Context) (T, error)) func(context.Context) (T, error) {
        return func(ctx context.Context) (T, error) {
            ctx, span := tracer.Start(ctx, name)
            defer span.End()
            
            start := time.Now()
            result, err := fn(ctx)
            duration := time.Since(start).Seconds()
            
            histogram.Observe(duration)
            if err != nil {
                span.RecordError(err)
                counter.Inc(map[string]string{"status": "error"})
            } else {
                counter.Inc(map[string]string{"status": "success"})
            }
            
            return result, err
        }
    }
}
```

**Validates: Requirements 14.5**

## Go 1.25 Synctest Integration

Deterministic concurrency testing pattern.

```go
import "testing/synctest"

// TestConcurrentCache demonstrates synctest usage.
func TestConcurrentCache(t *testing.T) {
    synctest.Run(func() {
        cache := NewCache[string, int](WithTTL(time.Second))
        
        // Concurrent writes
        var wg sync.WaitGroup
        for i := 0; i < 100; i++ {
            wg.Add(1)
            go func(n int) {
                defer wg.Done()
                cache.Set(fmt.Sprintf("key%d", n), n)
            }(i)
        }
        wg.Wait()
        
        // Verify all writes succeeded
        for i := 0; i < 100; i++ {
            val, ok := cache.Get(fmt.Sprintf("key%d", i))
            if !ok || val != i {
                t.Errorf("expected %d, got %d", i, val)
            }
        }
        
        // Advance time to trigger TTL expiry
        synctest.Wait()
        time.Sleep(2 * time.Second)
        synctest.Wait()
        
        // Verify expiry
        _, ok := cache.Get("key0")
        if ok {
            t.Error("expected key to be expired")
        }
    })
}
```

**Validates: Requirements 9.1, 9.2**
