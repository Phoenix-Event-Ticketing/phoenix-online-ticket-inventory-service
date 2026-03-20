package service

import (
	"context"
	"errors"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/repository"
)

// ErrNotImplemented indicates scaffold handlers without business logic yet.
var ErrNotImplemented = errors.New("not implemented")

// InventoryService coordinates inventory operations.
type InventoryService struct {
	repo *repository.InventoryRepository
}

// NewInventoryService creates the service.
func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

// CreateInventory is a placeholder for POST /inventory.
func (s *InventoryService) CreateInventory(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// BulkCreate is a placeholder for POST /inventory/bulk.
func (s *InventoryService) BulkCreate(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// Update is a placeholder for PUT /inventory/{inventoryId}.
func (s *InventoryService) Update(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// ListByEvent is a placeholder for GET /inventory/event/{eventId}.
func (s *InventoryService) ListByEvent(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// AvailabilitySummary is a placeholder for GET /inventory/event/{eventId}/availability.
func (s *InventoryService) AvailabilitySummary(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// GetByID is a placeholder for GET /inventory/{inventoryId}.
func (s *InventoryService) GetByID(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// Hold is a placeholder for POST /inventory/hold.
func (s *InventoryService) Hold(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// Confirm is a placeholder for POST /inventory/confirm.
func (s *InventoryService) Confirm(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// Release is a placeholder for POST /inventory/release.
func (s *InventoryService) Release(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}
