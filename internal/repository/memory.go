package repository

import (
	"context"
	"sort"
	"sync"
)

// DefaultPackSizes is the seed configuration used when an in-memory repository
// is created without explicit sizes.
var DefaultPackSizes = []int{250, 500, 1000, 2000, 5000}

// MemoryRepository is a thread-safe, in-memory implementation of
// PackSizeRepository. Data lives only for the lifetime of the process, which is
// sufficient for this application and keeps local/Heroku deployment dependency
// free. Concurrent access is guarded by a read/write mutex.
type MemoryRepository struct {
	mu    sync.RWMutex
	sizes []int
}

// NewMemoryRepository creates a repository seeded with DefaultPackSizes.
func NewMemoryRepository() *MemoryRepository {
	return NewMemoryRepositoryWithSizes(DefaultPackSizes)
}

// NewMemoryRepositoryWithSizes creates a repository seeded with the given sizes.
// The input is defensively copied so callers cannot mutate internal state.
func NewMemoryRepositoryWithSizes(sizes []int) *MemoryRepository {
	return &MemoryRepository{sizes: cloneSorted(sizes)}
}

// GetPackSizes returns a copy of the configured pack sizes (sorted ascending).
func (r *MemoryRepository) GetPackSizes(_ context.Context) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy so external mutation cannot corrupt the stored slice.
	out := make([]int, len(r.sizes))
	copy(out, r.sizes)
	return out, nil
}

// SetPackSizes replaces the stored pack sizes with a sorted copy of the input.
func (r *MemoryRepository) SetPackSizes(_ context.Context, sizes []int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sizes = cloneSorted(sizes)
	return nil
}

// cloneSorted returns a sorted copy of the input slice.
func cloneSorted(in []int) []int {
	out := make([]int, len(in))
	copy(out, in)
	sort.Ints(out)
	return out
}
