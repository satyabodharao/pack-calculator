// Package calculator contains the pure, dependency-free core logic that decides
// how many of each pack size should be shipped to fulfil a given order.
//
// The business rules, in priority order, are:
//
//	Rule 1: Only whole packs may be shipped (packs cannot be broken open).
//	Rule 2: Subject to Rule 1, ship the LEAST number of items possible.
//	Rule 3: Subject to Rules 1 & 2, ship the FEWEST number of packs possible.
//
// Rule 2 always takes precedence over Rule 3.
//
// This package performs no I/O and has no external dependencies, which keeps the
// algorithm easy to reason about and exhaustively unit-testable.
package calculator

import (
	"errors"
	"sort"
)

// ErrNoPackSizes is returned when Calculate is asked to fulfil an order but no
// pack sizes have been configured.
var ErrNoPackSizes = errors.New("no pack sizes configured")

// ErrInvalidPackSize is returned when a pack size is not a positive integer.
var ErrInvalidPackSize = errors.New("pack sizes must be positive integers")

// Pack represents a quantity of a particular pack size in the final shipment.
type Pack struct {
	Size  int `json:"size"`  // the pack size (e.g. 500)
	Count int `json:"count"` // how many packs of this size to ship
}

// Result is the outcome of a calculation: the packs to ship plus useful totals.
type Result struct {
	Packs      []Pack `json:"packs"`       // packs to ship, ordered largest size first
	TotalItems int    `json:"total_items"` // sum of items across all packs
	TotalPacks int    `json:"total_packs"` // sum of pack counts
}

// Calculate determines the optimal set of packs to ship for the given order.
//
// It uses a bounded dynamic-programming approach that is correct for arbitrary
// pack-size sets (including pathological ones such as {23, 31, 53}).
//
// Reasoning behind the upper bound used by the DP:
//
//	Let maxPack be the largest configured pack size. The optimal total number of
//	items T satisfies order <= T < order + maxPack. If T were >= order + maxPack
//	we could remove one pack of the largest size and STILL meet the order while
//	shipping fewer items — which would contradict Rule 2. Therefore we only ever
//	need to consider totals in the range [order, order+maxPack).
//
// Within that range:
//
//	dp[i]     = the minimum number of packs whose sizes sum to EXACTLY i
//	            (math.MaxInt sentinel means "i is unreachable").
//	choice[i] = a pack size used on an optimal path to i, for reconstruction.
//
// The smallest reachable i >= order is the answer to Rule 2 (fewest items), and
// dp[i] at that point is the answer to Rule 3 (fewest packs).
func Calculate(order int, packSizes []int) (Result, error) {
	// Normalise and validate the configured pack sizes first.
	sizes, err := normalisePackSizes(packSizes)
	if err != nil {
		return Result{}, err
	}

	// An order of zero (or negative) needs nothing shipped.
	if order <= 0 {
		return Result{Packs: []Pack{}, TotalItems: 0, TotalPacks: 0}, nil
	}

	maxPack := sizes[len(sizes)-1] // sizes is sorted ascending, so last is largest
	bound := order + maxPack       // exclusive-ish upper limit explained above

	// dp[i] holds the minimum pack count to make exactly i items.
	// We use len bound+1 so index `bound` itself is addressable.
	const unreachable = int(^uint(0) >> 1) // math.MaxInt without importing math
	dp := make([]int, bound+1)
	choice := make([]int, bound+1)
	for i := 1; i <= bound; i++ {
		dp[i] = unreachable
		choice[i] = -1
	}
	dp[0] = 0 // zero items needs zero packs

	// Classic unbounded-knapsack fill: for every reachable total i, try adding
	// one pack of each size and relax the resulting total.
	for i := 0; i <= bound; i++ {
		if dp[i] == unreachable {
			continue // i itself is not reachable, nothing to extend
		}
		for _, s := range sizes {
			next := i + s
			if next > bound {
				break // sizes is ascending, so all further sizes also overflow
			}
			if dp[i]+1 < dp[next] {
				dp[next] = dp[i] + 1
				choice[next] = s
			}
		}
	}

	// Rule 2: find the smallest total T >= order that is actually reachable.
	target := -1
	for i := order; i <= bound; i++ {
		if dp[i] != unreachable {
			target = i
			break
		}
	}
	if target == -1 {
		// Should be impossible given the bound, but guard defensively.
		return Result{}, ErrNoPackSizes
	}

	// Reconstruct the packs by walking the choice[] trail back to zero.
	counts := make(map[int]int)
	for t := target; t > 0; {
		s := choice[t]
		counts[s]++
		t -= s
	}

	return buildResult(counts), nil
}

// normalisePackSizes validates, de-duplicates and sorts (ascending) the pack
// sizes so the algorithm can rely on a clean, ordered slice.
func normalisePackSizes(packSizes []int) ([]int, error) {
	if len(packSizes) == 0 {
		return nil, ErrNoPackSizes
	}
	seen := make(map[int]struct{}, len(packSizes))
	out := make([]int, 0, len(packSizes))
	for _, s := range packSizes {
		if s <= 0 {
			return nil, ErrInvalidPackSize
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Ints(out)
	return out, nil
}

// buildResult turns a size->count map into a Result with packs ordered largest
// first (the conventional, human-friendly presentation) and computed totals.
func buildResult(counts map[int]int) Result {
	packs := make([]Pack, 0, len(counts))
	totalItems, totalPacks := 0, 0
	for size, count := range counts {
		packs = append(packs, Pack{Size: size, Count: count})
		totalItems += size * count
		totalPacks += count
	}
	sort.Slice(packs, func(i, j int) bool { return packs[i].Size > packs[j].Size })
	return Result{Packs: packs, TotalItems: totalItems, TotalPacks: totalPacks}
}
