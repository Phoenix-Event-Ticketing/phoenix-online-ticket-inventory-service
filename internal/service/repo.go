package service

import (
	"context"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
)

// InventoryRepo is the persistence layer used by InventoryService.
// Implemented by *repository.InventoryRepository and by NewFakeInventoryRepo.
type InventoryRepo interface {
	InsertInventory(ctx context.Context, doc *domain.Inventory) error
	InsertInventoryMany(ctx context.Context, docs []interface{}) error
	FindInventoryByID(ctx context.Context, inventoryID string) (*domain.Inventory, error)
	FindInventoriesByEvent(ctx context.Context, eventID string) ([]domain.Inventory, error)
	FindInventoryByEventAndType(ctx context.Context, eventID string, ticketType domain.TicketType) (*domain.Inventory, error)
	ReplaceInventory(ctx context.Context, doc *domain.Inventory) error
	ReserveHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error)
	RollbackReserve(ctx context.Context, inventoryID string, qty int) error
	ConfirmHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error)
	ReleaseHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error)
	ReplaceHold(ctx context.Context, h *domain.TicketHold) error
	FindHoldByBookingID(ctx context.Context, bookingID string) (*domain.TicketHold, error)
	CountInventoryByEventAndTypeExcluding(ctx context.Context, eventID string, ticketType domain.TicketType, excludeID string) (int64, error)
}
