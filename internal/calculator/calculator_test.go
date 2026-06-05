package calculator

import (
	"errors"
	"reflect"
	"testing"
)

// packsToMap is a test helper that flattens a Result's packs into a size->count
// map so assertions are independent of slice ordering.
func packsToMap(r Result) map[int]int {
	m := make(map[int]int)
	for _, p := range r.Packs {
		m[p.Size] = p.Count
	}
	return m
}

func TestCalculate_BriefExamples(t *testing.T) {
	defaultSizes := []int{250, 500, 1000, 2000, 5000}

	tests := []struct {
		name  string
		order int
		want  map[int]int
	}{
		{"one item", 1, map[int]int{250: 1}},
		{"exactly 250", 250, map[int]int{250: 1}},
		{"251 items", 251, map[int]int{500: 1}},
		{"501 items", 501, map[int]int{500: 1, 250: 1}},
		{"12001 items", 12001, map[int]int{5000: 2, 2000: 1, 250: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.order, defaultSizes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMap := packsToMap(got); !reflect.DeepEqual(gotMap, tt.want) {
				t.Errorf("Calculate(%d) packs = %v, want %v", tt.order, gotMap, tt.want)
			}
		})
	}
}

func TestCalculate_EdgeCases(t *testing.T) {
	defaultSizes := []int{250, 500, 1000, 2000, 5000}

	t.Run("zero order ships nothing", func(t *testing.T) {
		got, err := Calculate(0, defaultSizes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.TotalPacks != 0 || len(got.Packs) != 0 {
			t.Errorf("expected empty shipment, got %+v", got)
		}
	})

	t.Run("negative order ships nothing", func(t *testing.T) {
		got, err := Calculate(-5, defaultSizes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.TotalPacks != 0 {
			t.Errorf("expected empty shipment, got %+v", got)
		}
	})

	t.Run("exact multiple uses largest packs efficiently", func(t *testing.T) {
		got, err := Calculate(10000, defaultSizes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[int]int{5000: 2}
		if gotMap := packsToMap(got); !reflect.DeepEqual(gotMap, want) {
			t.Errorf("got %v, want %v", gotMap, want)
		}
	})

	t.Run("single pack size", func(t *testing.T) {
		got, err := Calculate(7, []int{3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 3*3=9 is the least total >= 7 reachable with size 3.
		want := map[int]int{3: 3}
		if gotMap := packsToMap(got); !reflect.DeepEqual(gotMap, want) {
			t.Errorf("got %v, want %v", gotMap, want)
		}
		if got.TotalItems != 9 {
			t.Errorf("total items = %d, want 9", got.TotalItems)
		}
	})
}

func TestCalculate_Totals(t *testing.T) {
	got, err := Calculate(501, []int{250, 500, 1000, 2000, 5000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 750 {
		t.Errorf("total items = %d, want 750", got.TotalItems)
	}
	if got.TotalPacks != 2 {
		t.Errorf("total packs = %d, want 2", got.TotalPacks)
	}
}

// TestCalculate_HardCase guards against naive greedy implementations. The set
// {23,31,53} ordering 500000 is the well-known case where greedy fails: the
// correct answer minimises items first (exactly 500000) then packs.
func TestCalculate_HardCase(t *testing.T) {
	got, err := Calculate(500000, []int{23, 31, 53})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The optimal solution ships exactly 500000 items (zero overage).
	if got.TotalItems != 500000 {
		t.Errorf("total items = %d, want 500000 (no overage)", got.TotalItems)
	}
	// And it does so in the fewest packs: 2x23 + 7x31 + 9429x53 = 9438 packs.
	if got.TotalPacks != 9438 {
		t.Errorf("total packs = %d, want 9438", got.TotalPacks)
	}
	// Sanity: the packs really do sum to the reported total.
	sum := 0
	for _, p := range got.Packs {
		sum += p.Size * p.Count
	}
	if sum != got.TotalItems {
		t.Errorf("packs sum to %d but TotalItems = %d", sum, got.TotalItems)
	}
}

func TestCalculate_Errors(t *testing.T) {
	t.Run("no pack sizes", func(t *testing.T) {
		_, err := Calculate(100, nil)
		if !errors.Is(err, ErrNoPackSizes) {
			t.Errorf("expected ErrNoPackSizes, got %v", err)
		}
	})

	t.Run("invalid pack size", func(t *testing.T) {
		_, err := Calculate(100, []int{250, 0, 500})
		if !errors.Is(err, ErrInvalidPackSize) {
			t.Errorf("expected ErrInvalidPackSize, got %v", err)
		}
	})

	t.Run("negative pack size", func(t *testing.T) {
		_, err := Calculate(100, []int{-10})
		if !errors.Is(err, ErrInvalidPackSize) {
			t.Errorf("expected ErrInvalidPackSize, got %v", err)
		}
	})
}

func TestCalculate_DuplicateSizesAreDeduped(t *testing.T) {
	got, err := Calculate(251, []int{250, 250, 500, 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[int]int{500: 1}
	if gotMap := packsToMap(got); !reflect.DeepEqual(gotMap, want) {
		t.Errorf("got %v, want %v", gotMap, want)
	}
}
