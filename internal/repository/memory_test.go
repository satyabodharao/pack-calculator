package repository

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

func TestMemoryRepository_SeedsDefaults(t *testing.T) {
	repo := NewMemoryRepository()
	got, err := repo.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, DefaultPackSizes) {
		t.Errorf("seed = %v, want %v", got, DefaultPackSizes)
	}
}

func TestMemoryRepository_SetAndGet(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Input is intentionally unsorted to verify the repo sorts on write.
	if err := repo.SetPackSizes(ctx, []int{300, 100, 200}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := repo.GetPackSizes(ctx)
	want := []int{100, 200, 300}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after set, got %v, want %v", got, want)
	}
}

func TestMemoryRepository_ReturnsCopy(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	got, _ := repo.GetPackSizes(ctx)
	got[0] = 99999 // mutate the returned slice

	again, _ := repo.GetPackSizes(ctx)
	if again[0] == 99999 {
		t.Error("mutating returned slice corrupted internal state")
	}
}

// TestMemoryRepository_Concurrent exercises the mutex; run with -race.
func TestMemoryRepository_Concurrent(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = repo.SetPackSizes(ctx, []int{n + 1, n + 2})
		}(i)
		go func() {
			defer wg.Done()
			_, _ = repo.GetPackSizes(ctx)
		}()
	}
	wg.Wait()
}
