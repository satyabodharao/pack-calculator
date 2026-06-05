// Package service is the application/business layer. It sits between the HTTP
// API and the data-access + calculator layers, orchestrating them and applying
// input validation and logging. It contains no HTTP or persistence specifics,
// which keeps it independently testable.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/satyabodharao/pack-calculator/internal/calculator"
	"github.com/satyabodharao/pack-calculator/internal/repository"
)

// ErrInvalidOrder is returned when the requested order quantity is invalid.
var ErrInvalidOrder = errors.New("order must be a positive integer")

// ErrInvalidPackSizes is returned when a supplied pack-size set is invalid.
var ErrInvalidPackSizes = errors.New("pack sizes must be a non-empty list of positive integers")

// PackService coordinates pack-size storage and the calculation algorithm.
type PackService struct {
	repo   repository.PackSizeRepository
	logger *slog.Logger
}

// New creates a PackService backed by the given repository and logger.
func New(repo repository.PackSizeRepository, logger *slog.Logger) *PackService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PackService{repo: repo, logger: logger}
}

// GetPackSizes returns the currently configured pack sizes.
func (s *PackService) GetPackSizes(ctx context.Context) ([]int, error) {
	sizes, err := s.repo.GetPackSizes(ctx)
	if err != nil {
		s.logger.Error("failed to read pack sizes", "error", err)
		return nil, fmt.Errorf("reading pack sizes: %w", err)
	}
	return sizes, nil
}

// UpdatePackSizes validates and persists a new set of pack sizes.
func (s *PackService) UpdatePackSizes(ctx context.Context, sizes []int) ([]int, error) {
	if len(sizes) == 0 {
		return nil, ErrInvalidPackSizes
	}
	for _, sz := range sizes {
		if sz <= 0 {
			return nil, ErrInvalidPackSizes
		}
	}

	if err := s.repo.SetPackSizes(ctx, sizes); err != nil {
		s.logger.Error("failed to persist pack sizes", "error", err)
		return nil, fmt.Errorf("saving pack sizes: %w", err)
	}
	s.logger.Info("pack sizes updated", "pack_sizes", sizes)

	// Return the canonical stored form (sorted, de-duplicated by the repo).
	return s.repo.GetPackSizes(ctx)
}

// Calculate computes the optimal packs for an order using the currently
// configured pack sizes.
func (s *PackService) Calculate(ctx context.Context, order int) (calculator.Result, error) {
	if order <= 0 {
		return calculator.Result{}, ErrInvalidOrder
	}

	sizes, err := s.repo.GetPackSizes(ctx)
	if err != nil {
		s.logger.Error("failed to read pack sizes for calculation", "error", err)
		return calculator.Result{}, fmt.Errorf("reading pack sizes: %w", err)
	}

	result, err := calculator.Calculate(order, sizes)
	if err != nil {
		s.logger.Error("calculation failed", "order", order, "pack_sizes", sizes, "error", err)
		return calculator.Result{}, err
	}

	s.logger.Info("calculation completed",
		"order", order,
		"total_items", result.TotalItems,
		"total_packs", result.TotalPacks,
	)
	return result, nil
}
