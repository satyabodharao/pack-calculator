package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/satyabodharao/pack-calculator/internal/repository"
)

func newTestService() *PackService {
	return New(repository.NewMemoryRepository(), nil)
}

func TestService_GetPackSizes(t *testing.T) {
	svc := newTestService()
	got, err := svc.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{250, 500, 1000, 2000, 5000}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestService_UpdatePackSizes(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	got, err := svc.UpdatePackSizes(ctx, []int{100, 50, 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{50, 100, 200} // repository sorts on write
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Confirm the change is reflected in a subsequent calculation.
	res, err := svc.Calculate(ctx, 51)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 100 {
		t.Errorf("after pack-size change, total items = %d, want 100", res.TotalItems)
	}
}

func TestService_UpdatePackSizes_Invalid(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	cases := map[string][]int{
		"empty":    {},
		"zero":     {250, 0},
		"negative": {-5},
	}
	for name, sizes := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpdatePackSizes(ctx, sizes)
			if !errors.Is(err, ErrInvalidPackSizes) {
				t.Errorf("expected ErrInvalidPackSizes, got %v", err)
			}
		})
	}
}

func TestService_Calculate(t *testing.T) {
	svc := newTestService()
	res, err := svc.Calculate(context.Background(), 12001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 12250 || res.TotalPacks != 4 {
		t.Errorf("got items=%d packs=%d, want items=12250 packs=4", res.TotalItems, res.TotalPacks)
	}
}

func TestService_Calculate_InvalidOrder(t *testing.T) {
	svc := newTestService()
	for _, order := range []int{0, -1} {
		if _, err := svc.Calculate(context.Background(), order); !errors.Is(err, ErrInvalidOrder) {
			t.Errorf("order %d: expected ErrInvalidOrder, got %v", order, err)
		}
	}
}
