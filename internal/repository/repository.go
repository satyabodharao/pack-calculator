// Package repository defines the data-access layer for pack sizes.
//
// The layer is expressed as an interface so the rest of the application depends
// only on behaviour, not on a concrete store. The current implementation is an
// in-memory store (see memory.go); a Postgres-backed implementation could be
// added later without changing the service or API layers.
package repository

import "context"

// PackSizeRepository is the data-access contract for persisting and retrieving
// the configurable set of pack sizes.
type PackSizeRepository interface {
	// GetPackSizes returns the currently configured pack sizes.
	GetPackSizes(ctx context.Context) ([]int, error)
	// SetPackSizes replaces the configured pack sizes with the supplied set.
	SetPackSizes(ctx context.Context, sizes []int) error
}
