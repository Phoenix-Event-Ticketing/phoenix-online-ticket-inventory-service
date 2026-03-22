package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

// fakeRepo is an in-memory implementation of InventoryRepo for tests and local tooling.
type fakeRepo struct {
	inv   map[string]*domain.Inventory
	holds map[string]*domain.TicketHold
	seq   int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		inv:   make(map[string]*domain.Inventory),
		holds: make(map[string]*domain.TicketHold),
	}
}

// NewFakeInventoryRepo returns an in-memory inventory repository (for handler tests and integration-style checks).
func NewFakeInventoryRepo() InventoryRepo {
	return newFakeRepo()
}

func (f *fakeRepo) dupKey() error {
	return mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000}}}
}

func (f *fakeRepo) hasEventType(eventID string, tt domain.TicketType, excludeID string) bool {
	for id, inv := range f.inv {
		if excludeID != "" && id == excludeID {
			continue
		}
		if inv.EventID == eventID && inv.TicketType == tt {
			return true
		}
	}
	return false
}

func (f *fakeRepo) InsertInventory(ctx context.Context, doc *domain.Inventory) error {
	if f.hasEventType(doc.EventID, doc.TicketType, "") {
		return f.dupKey()
	}
	if doc.InventoryID == "" {
		f.seq++
		doc.InventoryID = fmt.Sprintf("inv_%d", f.seq)
	}
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now
	cp := *doc
	f.inv[doc.InventoryID] = &cp
	return nil
}

func (f *fakeRepo) InsertInventoryMany(ctx context.Context, docs []interface{}) error {
	for _, d := range docs {
		di, ok := d.(domain.Inventory)
		if !ok {
			return fmt.Errorf("expected domain.Inventory")
		}
		if err := f.InsertInventory(ctx, &di); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeRepo) FindInventoryByID(ctx context.Context, inventoryID string) (*domain.Inventory, error) {
	v, ok := f.inv[inventoryID]
	if !ok {
		return nil, nil
	}
	cp := *v
	return &cp, nil
}

func (f *fakeRepo) FindInventoriesByEvent(ctx context.Context, eventID string) ([]domain.Inventory, error) {
	var out []domain.Inventory
	for _, v := range f.inv {
		if v.EventID == eventID {
			cp := *v
			out = append(out, cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) FindInventoryByEventAndType(ctx context.Context, eventID string, ticketType domain.TicketType) (*domain.Inventory, error) {
	for _, v := range f.inv {
		if v.EventID == eventID && v.TicketType == ticketType {
			cp := *v
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) ReplaceInventory(ctx context.Context, doc *domain.Inventory) error {
	if _, ok := f.inv[doc.InventoryID]; !ok {
		return mongo.ErrNoDocuments
	}
	doc.UpdatedAt = time.Now().UTC()
	cp := *doc
	f.inv[doc.InventoryID] = &cp
	return nil
}

func (f *fakeRepo) ReserveHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	inv, ok := f.inv[inventoryID]
	if !ok || inv.AvailableQuantity < qty {
		return nil, nil
	}
	inv.AvailableQuantity -= qty
	inv.HeldQuantity += qty
	cp := *inv
	return &cp, nil
}

func (f *fakeRepo) RollbackReserve(ctx context.Context, inventoryID string, qty int) error {
	if qty <= 0 {
		return nil
	}
	inv, ok := f.inv[inventoryID]
	if !ok {
		return nil
	}
	inv.AvailableQuantity += qty
	inv.HeldQuantity -= qty
	return nil
}

func (f *fakeRepo) ConfirmHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	inv, ok := f.inv[inventoryID]
	if !ok || inv.HeldQuantity < qty {
		return nil, nil
	}
	inv.HeldQuantity -= qty
	inv.SoldQuantity += qty
	cp := *inv
	return &cp, nil
}

func (f *fakeRepo) ReleaseHeld(ctx context.Context, inventoryID string, qty int) (*domain.Inventory, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	inv, ok := f.inv[inventoryID]
	if !ok || inv.HeldQuantity < qty {
		return nil, nil
	}
	inv.HeldQuantity -= qty
	inv.AvailableQuantity += qty
	cp := *inv
	return &cp, nil
}

func (f *fakeRepo) ReplaceHold(ctx context.Context, h *domain.TicketHold) error {
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now().UTC()
	}
	cp := *h
	f.holds[h.BookingID] = &cp
	return nil
}

func (f *fakeRepo) FindHoldByBookingID(ctx context.Context, bookingID string) (*domain.TicketHold, error) {
	h, ok := f.holds[bookingID]
	if !ok {
		return nil, nil
	}
	cp := *h
	return &cp, nil
}

func (f *fakeRepo) CountInventoryByEventAndTypeExcluding(ctx context.Context, eventID string, ticketType domain.TicketType, excludeID string) (int64, error) {
	var n int64
	for id, inv := range f.inv {
		if excludeID != "" && id == excludeID {
			continue
		}
		if inv.EventID == eventID && inv.TicketType == ticketType {
			n++
		}
	}
	return n, nil
}
