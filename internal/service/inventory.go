package service

import (
	"context"
	"strings"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

// InventoryService coordinates inventory operations.
type InventoryService struct {
	repo    InventoryRepo
	holdTTL time.Duration
}

// NewInventoryService creates the service.
func NewInventoryService(repo InventoryRepo, holdTTL time.Duration) *InventoryService {
	return &InventoryService{repo: repo, holdTTL: holdTTL}
}

// CreateInventory handles POST /inventory.
func (s *InventoryService) CreateInventory(ctx context.Context, req domain.CreateInventoryRequest) (*domain.Inventory, error) {
	if err := domain.ValidTicketType(req.TicketType); err != nil {
		return nil, err
	}

	doc := &domain.Inventory{
		EventID:           strings.TrimSpace(req.EventID),
		TicketType:        req.TicketType,
		Price:             req.Price,
		TotalQuantity:     req.TotalQuantity,
		HeldQuantity:      0,
		SoldQuantity:      0,
		AvailableQuantity: req.TotalQuantity,
	}
	if err := s.repo.InsertInventory(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateTicket
		}
		return nil, err
	}
	return doc, nil
}

// BulkCreate handles POST /inventory/bulk.
func (s *InventoryService) BulkCreate(ctx context.Context, req domain.BulkCreateRequest) ([]domain.Inventory, error) {
	eventID := strings.TrimSpace(req.EventID)
	seen := make(map[domain.TicketType]struct{})
	now := time.Now().UTC()
	docs := make([]interface{}, 0, len(req.Items))
	out := make([]domain.Inventory, 0, len(req.Items))

	for _, it := range req.Items {
		if err := domain.ValidTicketType(it.TicketType); err != nil {
			return nil, err
		}
		if _, dup := seen[it.TicketType]; dup {
			return nil, ErrDuplicateTicket
		}
		seen[it.TicketType] = struct{}{}

		doc := domain.Inventory{
			InventoryID:       repository.GenerateInventoryID(),
			EventID:           eventID,
			TicketType:        it.TicketType,
			Price:             it.Price,
			TotalQuantity:     it.TotalQuantity,
			HeldQuantity:      0,
			SoldQuantity:      0,
			AvailableQuantity: it.TotalQuantity,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		docs = append(docs, doc)
		out = append(out, doc)
	}

	if err := s.repo.InsertInventoryMany(ctx, docs); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateTicket
		}
		return nil, err
	}
	return out, nil
}

// UpdateInventory handles PUT /inventory/{inventoryId}.
func (s *InventoryService) UpdateInventory(ctx context.Context, inventoryID string, req domain.UpdateInventoryRequest) (*domain.Inventory, error) {
	if req.TicketType == nil && req.Price == nil && req.TotalQuantity == nil {
		return nil, ErrConflict
	}

	cur, err := s.repo.FindInventoryByID(ctx, inventoryID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrNotFound
	}

	if req.TicketType != nil {
		if err := domain.ValidTicketType(*req.TicketType); err != nil {
			return nil, err
		}
		n, err := s.repo.CountInventoryByEventAndTypeExcluding(ctx, cur.EventID, *req.TicketType, inventoryID)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			return nil, ErrDuplicateTicket
		}
		cur.TicketType = *req.TicketType
	}
	if req.Price != nil {
		cur.Price = *req.Price
	}
	if req.TotalQuantity != nil {
		cur.TotalQuantity = *req.TotalQuantity
	}

	cur.AvailableQuantity = cur.TotalQuantity - cur.SoldQuantity - cur.HeldQuantity
	if cur.AvailableQuantity < 0 {
		return nil, ErrConflict
	}

	if err := s.repo.ReplaceInventory(ctx, cur); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateTicket
		}
		return nil, err
	}
	return cur, nil
}

// ListByEvent handles GET /inventory/event/{eventId}.
func (s *InventoryService) ListByEvent(ctx context.Context, eventID string) ([]domain.Inventory, error) {
	return s.repo.FindInventoriesByEvent(ctx, eventID)
}

// AvailabilitySummary handles GET /inventory/event/{eventId}/availability.
func (s *InventoryService) AvailabilitySummary(ctx context.Context, eventID string) (*domain.AvailabilitySummary, error) {
	list, err := s.repo.FindInventoriesByEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.AvailabilityItem, 0, len(list))
	for _, inv := range list {
		items = append(items, domain.AvailabilityItem{
			InventoryID:       inv.InventoryID,
			TicketType:        inv.TicketType,
			Price:             inv.Price,
			TotalQuantity:     inv.TotalQuantity,
			HeldQuantity:      inv.HeldQuantity,
			SoldQuantity:      inv.SoldQuantity,
			AvailableQuantity: inv.AvailableQuantity,
		})
	}
	return &domain.AvailabilitySummary{EventID: eventID, Items: items}, nil
}

// GetByID handles GET /inventory/{inventoryId}.
func (s *InventoryService) GetByID(ctx context.Context, inventoryID string) (*domain.Inventory, error) {
	inv, err := s.repo.FindInventoryByID(ctx, inventoryID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrNotFound
	}
	return inv, nil
}

// Hold handles POST /inventory/hold.
func (s *InventoryService) Hold(ctx context.Context, req domain.HoldRequest, now time.Time) (*domain.HoldResponse, error) {
	if err := domain.ValidTicketType(req.TicketType); err != nil {
		return nil, err
	}
	bookingID := strings.TrimSpace(req.BookingID)
	eventID := strings.TrimSpace(req.EventID)

	inv, err := s.repo.FindInventoryByEventAndType(ctx, eventID, req.TicketType)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrNotFound
	}

	existing, err := s.repo.FindHoldByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		switch existing.Status {
		case domain.HoldConfirmed:
			return nil, ErrInvalidHoldState
		case domain.HoldHeld:
			if now.After(existing.ExpiresAt) {
				if _, relErr := s.repo.ReleaseHeld(ctx, existing.InventoryID, existing.Quantity); relErr != nil {
					return nil, relErr
				}
				existing.Status = domain.HoldReleased
				if err := s.repo.ReplaceHold(ctx, existing); err != nil {
					return nil, err
				}
				existing = nil
			} else {
				if existing.EventID == eventID && existing.TicketType == req.TicketType && existing.Quantity == req.Quantity && existing.InventoryID == inv.InventoryID {
					return &domain.HoldResponse{
						BookingID:  bookingID,
						HoldStatus: domain.HoldHeld,
						ExpiresAt:  existing.ExpiresAt,
					}, nil
				}
				return nil, ErrHoldParamsMismatch
			}
		case domain.HoldReleased:
			existing = nil
		default:
			return nil, ErrInvalidHoldState
		}
	}

	after, err := s.repo.ReserveHeld(ctx, inv.InventoryID, req.Quantity)
	if err != nil {
		return nil, err
	}
	if after == nil {
		return nil, ErrInsufficientStock
	}

	expires := now.Add(s.holdTTL).UTC()
	hold := domain.TicketHold{
		BookingID:   bookingID,
		InventoryID: inv.InventoryID,
		EventID:     eventID,
		TicketType:  req.TicketType,
		Quantity:    req.Quantity,
		Status:      domain.HoldHeld,
		ExpiresAt:   expires,
	}
	if err := s.repo.ReplaceHold(ctx, &hold); err != nil {
		_ = s.repo.RollbackReserve(ctx, inv.InventoryID, req.Quantity)
		return nil, err
	}

	return &domain.HoldResponse{
		BookingID:  bookingID,
		HoldStatus: domain.HoldHeld,
		ExpiresAt:  expires,
	}, nil
}

// Confirm handles POST /inventory/confirm.
func (s *InventoryService) Confirm(ctx context.Context, bookingID string, now time.Time) (*domain.TicketHold, error) {
	bookingID = strings.TrimSpace(bookingID)
	h, err := s.repo.FindHoldByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, ErrHoldNotFound
	}

	if h.Status == domain.HoldConfirmed {
		return h, nil
	}
	if h.Status != domain.HoldHeld {
		return nil, ErrInvalidHoldState
	}

	if now.After(h.ExpiresAt) {
		if _, relErr := s.repo.ReleaseHeld(ctx, h.InventoryID, h.Quantity); relErr != nil {
			return nil, relErr
		}
		h.Status = domain.HoldReleased
		if err := s.repo.ReplaceHold(ctx, h); err != nil {
			return nil, err
		}
		return nil, ErrHoldExpired
	}

	after, err := s.repo.ConfirmHeld(ctx, h.InventoryID, h.Quantity)
	if err != nil {
		return nil, err
	}
	if after == nil {
		return nil, ErrInsufficientStock
	}

	h.Status = domain.HoldConfirmed
	if err := s.repo.ReplaceHold(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

// Release handles POST /inventory/release.
func (s *InventoryService) Release(ctx context.Context, bookingID string) (*domain.TicketHold, error) {
	bookingID = strings.TrimSpace(bookingID)
	h, err := s.repo.FindHoldByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, ErrHoldNotFound
	}

	if h.Status == domain.HoldReleased {
		return h, nil
	}
	if h.Status != domain.HoldHeld {
		return nil, ErrInvalidHoldState
	}

	if _, err := s.repo.ReleaseHeld(ctx, h.InventoryID, h.Quantity); err != nil {
		return nil, err
	}

	h.Status = domain.HoldReleased
	if err := s.repo.ReplaceHold(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}
