// Package patterns provides generic design patterns for Go applications.
package patterns

import (
	"context"

	"github.com/authcorp/libs/go/src/functional"
)

// Repository defines generic CRUD operations with type safety.
// T is the entity type, ID is the identifier type (must be comparable).
type Repository[T any, ID comparable] interface {
	// Get retrieves an entity by ID, returning Option for type-safe null handling.
	Get(ctx context.Context, id ID) functional.Option[T]

	// Save persists an entity and returns the saved entity.
	Save(ctx context.Context, entity T) functional.Result[T]

	// Delete removes an entity by ID.
	Delete(ctx context.Context, id ID) error

	// List returns all entities.
	List(ctx context.Context) functional.Result[[]T]

	// Exists checks if an entity exists by ID.
	Exists(ctx context.Context, id ID) bool
}

// ReadRepository defines read-only repository operations.
type ReadRepository[T any, ID comparable] interface {
	Get(ctx context.Context, id ID) functional.Option[T]
	List(ctx context.Context) functional.Result[[]T]
	Exists(ctx context.Context, id ID) bool
}

// WriteRepository defines write-only repository operations.
type WriteRepository[T any, ID comparable] interface {
	Save(ctx context.Context, entity T) functional.Result[T]
	Delete(ctx context.Context, id ID) error
}

// PagedRepository extends Repository with pagination support.
type PagedRepository[T any, ID comparable] interface {
	Repository[T, ID]

	// ListPaged returns a page of entities.
	ListPaged(ctx context.Context, page, pageSize int) functional.Result[Page[T]]

	// Count returns the total number of entities.
	Count(ctx context.Context) (int64, error)
}

// Page represents a page of results.
type Page[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// NewPage creates a new page of results.
func NewPage[T any](items []T, page, pageSize int, totalItems int64) Page[T] {
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}
	return Page[T]{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

// HasNext returns true if there are more pages.
func (p Page[T]) HasNext() bool {
	return p.Page < p.TotalPages
}

// HasPrev returns true if there are previous pages.
func (p Page[T]) HasPrev() bool {
	return p.Page > 1
}

// IsEmpty returns true if the page has no items.
func (p Page[T]) IsEmpty() bool {
	return len(p.Items) == 0
}

// IDExtractor extracts the ID from an entity.
type IDExtractor[T any, ID comparable] func(entity T) ID

// InMemoryRepository is a simple in-memory implementation for testing.
type InMemoryRepository[T any, ID comparable] struct {
	data      map[ID]T
	extractor IDExtractor[T, ID]
}

// NewInMemoryRepository creates a new in-memory repository.
func NewInMemoryRepository[T any, ID comparable](extractor IDExtractor[T, ID]) *InMemoryRepository[T, ID] {
	return &InMemoryRepository[T, ID]{
		data:      make(map[ID]T),
		extractor: extractor,
	}
}

// Get retrieves an entity by ID.
func (r *InMemoryRepository[T, ID]) Get(ctx context.Context, id ID) functional.Option[T] {
	if entity, ok := r.data[id]; ok {
		return functional.Some(entity)
	}
	return functional.None[T]()
}

// Save persists an entity.
func (r *InMemoryRepository[T, ID]) Save(ctx context.Context, entity T) functional.Result[T] {
	id := r.extractor(entity)
	r.data[id] = entity
	return functional.Ok(entity)
}

// Delete removes an entity by ID.
func (r *InMemoryRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	delete(r.data, id)
	return nil
}

// List returns all entities.
func (r *InMemoryRepository[T, ID]) List(ctx context.Context) functional.Result[[]T] {
	entities := make([]T, 0, len(r.data))
	for _, entity := range r.data {
		entities = append(entities, entity)
	}
	return functional.Ok(entities)
}

// Exists checks if an entity exists.
func (r *InMemoryRepository[T, ID]) Exists(ctx context.Context, id ID) bool {
	_, ok := r.data[id]
	return ok
}

// Clear removes all entities (useful for testing).
func (r *InMemoryRepository[T, ID]) Clear() {
	r.data = make(map[ID]T)
}

// Size returns the number of entities.
func (r *InMemoryRepository[T, ID]) Size() int {
	return len(r.data)
}

// Ensure InMemoryRepository implements Repository.
var _ Repository[any, string] = (*InMemoryRepository[any, string])(nil)
