# Events Module

Generic event bus with filtering and retry support.

## EventBus[E]

Publish/subscribe event system.

```go
type UserCreated struct {
    UserID string
    Email  string
}

func (e UserCreated) Type() string { return "user.created" }

bus := events.NewEventBus[UserCreated]()

// Subscribe to events
sub := bus.Subscribe("user.created", func(ctx context.Context, e UserCreated) error {
    log.Printf("User created: %s", e.UserID)
    return nil
})

// Publish event
bus.Publish(ctx, UserCreated{UserID: "123", Email: "[email]"})

// Cancel subscription
sub.Cancel()
```

### Filtered Subscriptions

```go
// Only receive events matching filter
bus.SubscribeWithFilter("user.created",
    func(ctx context.Context, e UserCreated) error {
        return sendWelcomeEmail(e.Email)
    },
    func(e UserCreated) bool {
        return strings.HasSuffix(e.Email, "@company.com")
    },
)
```

### Async Event Bus

```go
// Events delivered asynchronously
asyncBus := events.NewAsyncEventBus[UserCreated]()

asyncBus.Subscribe("user.created", func(ctx context.Context, e UserCreated) error {
    // Handler runs in goroutine
    return processUser(e)
})
```

### Publishing Multiple Events

```go
events := []UserCreated{
    {UserID: "1", Email: "[email]"},
    {UserID: "2", Email: "[email]"},
}

// Publish all events (stops on first error)
err := bus.PublishAll(ctx, events)
```

### Checking Subscribers

```go
if bus.HasSubscribers("user.created") {
    bus.Publish(ctx, event)
}

count := bus.SubscriberCount("user.created")
```

### Clearing Subscriptions

```go
// Remove all subscriptions
bus.Clear()
```
