package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
)

type stubEventVerifier struct {
	exists bool
	err    error
}

func (s stubEventVerifier) EventExists(ctx context.Context, eventID string) (bool, error) {
	return s.exists, s.err
}

func TestCreateInventory_InvalidTicketType(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	_, err := s.CreateInventory(context.Background(), domain.CreateInventoryRequest{
		EventID: "e1", TicketType: "BAD", Price: 1, TotalQuantity: 1,
	})
	if !errors.Is(err, domain.ErrInvalidTicketType) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateInventory_DuplicateKey(t *testing.T) {
	r := newFakeRepo()
	s := NewInventoryService(r, time.Minute)
	_, err := s.CreateInventory(context.Background(), domain.CreateInventoryRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateInventory(context.Background(), domain.CreateInventoryRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Price: 2, TotalQuantity: 5,
	})
	if !errors.Is(err, ErrDuplicateTicket) {
		t.Fatalf("expected ErrDuplicateTicket, got %v", err)
	}
}

func TestBulkCreate_DuplicateInRequest(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	_, err := s.BulkCreate(context.Background(), domain.BulkCreateRequest{
		EventID: "e1",
		Items: []domain.BulkInventoryItem{
			{TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 1},
			{TicketType: domain.TicketVIP, Price: 2, TotalQuantity: 1},
		},
	})
	if !errors.Is(err, ErrDuplicateTicket) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateInventory_NoOpBody(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	_, err := s.UpdateInventory(context.Background(), "inv_x", domain.UpdateInventoryRequest{})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateInventory_NotFound(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	p := domain.TicketVIP
	_, err := s.UpdateInventory(context.Background(), "missing", domain.UpdateInventoryRequest{TicketType: &p})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 3, AvailableQuantity: 3,
	})
	s := NewInventoryService(r, time.Minute)
	inv, err := s.GetByID(context.Background(), "i1")
	if err != nil || inv.InventoryID != "i1" {
		t.Fatalf("err=%v inv=%+v", err, inv)
	}
}

func TestHold_IdempotentSameParams(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 10,
	})
	s := NewInventoryService(r, time.Minute)
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	req := domain.HoldRequest{EventID: "e1", TicketType: domain.TicketVIP, Quantity: 2, BookingID: "b1"}
	a, err := s.Hold(context.Background(), req, now)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Hold(context.Background(), req, now)
	if err != nil {
		t.Fatal(err)
	}
	if a.HoldStatus != b.HoldStatus || !a.ExpiresAt.Equal(b.ExpiresAt) {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
}

func TestUpdateInventory_InvalidTicketType(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 10,
	})
	s := NewInventoryService(r, time.Minute)
	bad := domain.TicketType("BAD")
	_, err := s.UpdateInventory(context.Background(), "i1", domain.UpdateInventoryRequest{TicketType: &bad})
	if !errors.Is(err, domain.ErrInvalidTicketType) {
		t.Fatalf("got %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	_, err := s.GetByID(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestAvailabilitySummary(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Price: 10, TotalQuantity: 5, HeldQuantity: 1, SoldQuantity: 1, AvailableQuantity: 3,
	})
	s := NewInventoryService(r, time.Minute)
	sum, err := s.AvailabilitySummary(context.Background(), "e1")
	if err != nil {
		t.Fatal(err)
	}
	if sum.EventID != "e1" || len(sum.Items) != 1 || sum.Items[0].AvailableQuantity != 3 {
		t.Fatalf("unexpected %+v", sum)
	}
}

func TestHoldConfirmRelease(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Price: 10, TotalQuantity: 10, AvailableQuantity: 10,
	})
	s := NewInventoryService(r, time.Minute)
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	resp, err := s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 2, BookingID: "b1",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if resp.HoldStatus != domain.HoldHeld {
		t.Fatalf("hold: %+v", resp)
	}

	h, err := s.Confirm(context.Background(), "b1", now)
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != domain.HoldConfirmed {
		t.Fatalf("confirm: %+v", h)
	}

	r2 := newFakeRepo()
	_ = r2.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Price: 10, TotalQuantity: 10, AvailableQuantity: 10,
	})
	s2 := NewInventoryService(r2, time.Minute)
	_, err = s2.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 1, BookingID: "b2",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := s2.Release(context.Background(), "b2")
	if err != nil {
		t.Fatal(err)
	}
	if h2.Status != domain.HoldReleased {
		t.Fatalf("release: %+v", h2)
	}
}

func TestHold_InsufficientStock(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketStandard,
		Price: 5, TotalQuantity: 1, AvailableQuantity: 1,
	})
	s := NewInventoryService(r, time.Minute)
	_, err := s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketStandard, Quantity: 3, BookingID: "b1",
	}, time.Now())
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("got %v", err)
	}
}

func TestConfirm_NotFound(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	_, err := s.Confirm(context.Background(), "missing", time.Now())
	if !errors.Is(err, ErrHoldNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestConfirm_AlreadyConfirmedIdempotent(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 8, HeldQuantity: 2,
	})
	_ = r.ReplaceHold(context.Background(), &domain.TicketHold{
		BookingID: "b1", InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 2, Status: domain.HoldConfirmed, ExpiresAt: time.Now().Add(time.Hour),
	})
	s := NewInventoryService(r, time.Minute)
	h, err := s.Confirm(context.Background(), "b1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != domain.HoldConfirmed {
		t.Fatal(h.Status)
	}
}

func TestConfirm_ExpiredHold(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 8, HeldQuantity: 2,
	})
	past := time.Now().Add(-time.Hour)
	_ = r.ReplaceHold(context.Background(), &domain.TicketHold{
		BookingID: "b1", InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 2, Status: domain.HoldHeld, ExpiresAt: past,
	})
	s := NewInventoryService(r, time.Minute)
	_, err := s.Confirm(context.Background(), "b1", time.Now())
	if !errors.Is(err, ErrHoldExpired) {
		t.Fatalf("got %v", err)
	}
}

func TestRelease_ConfirmedHold(t *testing.T) {
	r := newFakeRepo()
	_ = r.ReplaceHold(context.Background(), &domain.TicketHold{
		BookingID: "b1", InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 1, Status: domain.HoldConfirmed, ExpiresAt: time.Now().Add(time.Hour),
	})
	s := NewInventoryService(r, time.Minute)
	_, err := s.Release(context.Background(), "b1")
	if !errors.Is(err, ErrInvalidHoldState) {
		t.Fatalf("got %v", err)
	}
}

func TestRelease_AlreadyReleased(t *testing.T) {
	r := newFakeRepo()
	_ = r.ReplaceHold(context.Background(), &domain.TicketHold{
		BookingID: "b1", InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 1, Status: domain.HoldReleased, ExpiresAt: time.Now().Add(time.Hour),
	})
	s := NewInventoryService(r, time.Minute)
	h, err := s.Release(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != domain.HoldReleased {
		t.Fatal(h.Status)
	}
}

func TestListByEvent(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 5, AvailableQuantity: 5,
	})
	s := NewInventoryService(r, time.Minute)
	list, err := s.ListByEvent(context.Background(), "e1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list err=%v len=%d", err, len(list))
	}
}

func TestHold_InventoryNotFound(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	_, err := s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 1, BookingID: "b1",
	}, time.Now())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateInventory_DuplicateTicketType(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 10,
	})
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i2", EventID: "e1", TicketType: domain.TicketStandard,
		TotalQuantity: 5, AvailableQuantity: 5,
	})
	s := NewInventoryService(r, time.Minute)
	st := domain.TicketStandard
	_, err := s.UpdateInventory(context.Background(), "i1", domain.UpdateInventoryRequest{TicketType: &st})
	if !errors.Is(err, ErrDuplicateTicket) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateInventory_TotalQuantityConflict(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, SoldQuantity: 4, HeldQuantity: 4, AvailableQuantity: 2,
	})
	s := NewInventoryService(r, time.Minute)
	tq := 7
	_, err := s.UpdateInventory(context.Background(), "i1", domain.UpdateInventoryRequest{TotalQuantity: &tq})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateInventory_Price(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Price: 10, TotalQuantity: 10, AvailableQuantity: 10,
	})
	s := NewInventoryService(r, time.Minute)
	price := 20.0
	inv, err := s.UpdateInventory(context.Background(), "i1", domain.UpdateInventoryRequest{Price: &price})
	if err != nil || inv.Price != 20 {
		t.Fatalf("err=%v inv=%+v", err, inv)
	}
}

func TestHold_UnknownHoldStatus(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 10,
	})
	_ = r.ReplaceHold(context.Background(), &domain.TicketHold{
		BookingID: "b1", InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 1, Status: domain.HoldStatus("UNKNOWN"), ExpiresAt: time.Now().Add(time.Hour),
	})
	s := NewInventoryService(r, time.Minute)
	_, err := s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 1, BookingID: "b1",
	}, time.Now())
	if !errors.Is(err, ErrInvalidHoldState) {
		t.Fatalf("got %v", err)
	}
}

func TestHold_ExistingConfirmed(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 10,
	})
	_ = r.ReplaceHold(context.Background(), &domain.TicketHold{
		BookingID: "b1", InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 1, Status: domain.HoldConfirmed, ExpiresAt: time.Now().Add(time.Hour),
	})
	s := NewInventoryService(r, time.Minute)
	_, err := s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 1, BookingID: "b1",
	}, time.Now())
	if !errors.Is(err, ErrInvalidHoldState) {
		t.Fatalf("got %v", err)
	}
}

func TestHold_ParamsMismatch(t *testing.T) {
	r := newFakeRepo()
	_ = r.InsertInventory(context.Background(), &domain.Inventory{
		InventoryID: "i1", EventID: "e1", TicketType: domain.TicketVIP,
		TotalQuantity: 10, AvailableQuantity: 10,
	})
	s := NewInventoryService(r, time.Minute)
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 2, BookingID: "b1",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Hold(context.Background(), domain.HoldRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Quantity: 3, BookingID: "b1",
	}, now)
	if !errors.Is(err, ErrHoldParamsMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestBulkCreate_SingleRow(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute)
	out, err := s.BulkCreate(context.Background(), domain.BulkCreateRequest{
		EventID: "e1",
		Items: []domain.BulkInventoryItem{
			{TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].InventoryID == "" {
		t.Fatalf("unexpected %+v", out[0])
	}
}

func TestCreateInventory_EventNotFound(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute, stubEventVerifier{exists: false})
	_, err := s.CreateInventory(context.Background(), domain.CreateInventoryRequest{
		EventID: "missing", TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 1,
	})
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateInventory_EventServiceUnavailable(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute, stubEventVerifier{err: fmt.Errorf("boom")})
	_, err := s.CreateInventory(context.Background(), domain.CreateInventoryRequest{
		EventID: "e1", TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 1,
	})
	if !errors.Is(err, ErrEventServiceUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestBulkCreate_EventNotFound(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute, stubEventVerifier{exists: false})
	_, err := s.BulkCreate(context.Background(), domain.BulkCreateRequest{
		EventID: "missing",
		Items: []domain.BulkInventoryItem{
			{TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 1},
		},
	})
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestBulkCreate_EventServiceUnavailable(t *testing.T) {
	s := NewInventoryService(newFakeRepo(), time.Minute, stubEventVerifier{err: fmt.Errorf("boom")})
	_, err := s.BulkCreate(context.Background(), domain.BulkCreateRequest{
		EventID: "e1",
		Items: []domain.BulkInventoryItem{
			{TicketType: domain.TicketVIP, Price: 1, TotalQuantity: 1},
		},
	})
	if !errors.Is(err, ErrEventServiceUnavailable) {
		t.Fatalf("got %v", err)
	}
}
