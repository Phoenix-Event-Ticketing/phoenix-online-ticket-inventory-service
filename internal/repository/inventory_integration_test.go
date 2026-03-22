package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func mongoURIFromEnv() string {
	if u := os.Getenv("MONGODB_URI"); u != "" {
		return u
	}
	return "mongodb://127.0.0.1:27017"
}

func TestInventoryRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test (-short)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURIFromEnv()))
	if err != nil {
		t.Skipf("mongo not available: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	dbName := fmt.Sprintf("repo_it_%d", time.Now().UnixNano())
	db := client.Database(dbName)
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
	})
	repo := NewInventoryRepository(db)
	if err := repo.Ping(ctx); err != nil {
		t.Skipf("mongo ping: %v", err)
	}
	if err := repo.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertInventory(ctx, &domain.Inventory{
		EventID:           "e1",
		TicketType:        domain.TicketVIP,
		Price:             99,
		TotalQuantity:     10,
		HeldQuantity:      0,
		SoldQuantity:      0,
		AvailableQuantity: 10,
	}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.FindInventoriesByEvent(ctx, "e1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list err=%v len=%d", err, len(list))
	}
	inv, err := repo.FindInventoryByEventAndType(ctx, "e1", domain.TicketVIP)
	if err != nil || inv == nil {
		t.Fatal(err)
	}
	byID, err := repo.FindInventoryByID(ctx, inv.InventoryID)
	if err != nil || byID == nil {
		t.Fatal(err)
	}
	n, err := repo.CountInventoryByEventAndTypeExcluding(ctx, "e1", domain.TicketVIP, "")
	if err != nil || n != 1 {
		t.Fatalf("count err=%v n=%d", err, n)
	}
	after, err := repo.ReserveHeld(ctx, inv.InventoryID, 2)
	if err != nil || after == nil {
		t.Fatal(err)
	}
	if err := repo.RollbackReserve(ctx, inv.InventoryID, 2); err != nil {
		t.Fatal(err)
	}
	after, err = repo.ReserveHeld(ctx, inv.InventoryID, 2)
	if err != nil || after == nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceHold(ctx, &domain.TicketHold{
		BookingID: "b1", InventoryID: inv.InventoryID, EventID: "e1", TicketType: domain.TicketVIP,
		Quantity: 2, Status: domain.HoldHeld, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	h, err := repo.FindHoldByBookingID(ctx, "b1")
	if err != nil || h == nil {
		t.Fatal(err)
	}
	after, err = repo.ConfirmHeld(ctx, inv.InventoryID, 2)
	if err != nil || after == nil {
		t.Fatal(err)
	}
	if err := repo.InsertInventoryMany(ctx, []interface{}{
		domain.Inventory{
			InventoryID: GenerateInventoryID(), EventID: "e2", TicketType: domain.TicketStandard,
			Price: 10, TotalQuantity: 5, AvailableQuantity: 5,
		},
	}); err != nil {
		t.Fatal(err)
	}
	_, err = repo.ReleaseHeld(ctx, inv.InventoryID, 0)
	if err == nil {
		t.Fatal("expected error for qty 0")
	}
}
