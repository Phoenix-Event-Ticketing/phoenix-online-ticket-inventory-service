package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/handler"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/repository"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func mongoURI() string {
	if u := os.Getenv("MONGODB_URI"); u != "" {
		return u
	}
	return "mongodb://127.0.0.1:27017"
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test (-short)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI()))
	if err != nil {
		t.Fatalf("mongo connect: %v", err)
	}

	dbName := fmt.Sprintf("inv_it_%d_%s", time.Now().UnixNano(), sanitizeDBName(t.Name()))
	db := client.Database(dbName)
	repo := repository.NewInventoryRepository(db)
	if err := repo.EnsureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		t.Fatalf("ensure indexes: %v", err)
	}

	svc := service.NewInventoryService(repo, 30*time.Minute)
	invHandler := handler.NewInventoryHandler(svc)
	router := handler.NewRouter(zap.NewNop(), invHandler)
	srv := httptest.NewServer(router)

	t.Cleanup(func() {
		srv.Close()
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return srv
}

func sanitizeDBName(name string) string {
	b := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b = append(b, r)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

func TestInventoryHoldConfirmFlow(t *testing.T) {
	srv := newTestServer(t)
	eventID := fmt.Sprintf("evt_flow_%d", time.Now().UnixNano())

	// Create inventory
	createBody := domain.CreateInventoryRequest{
		EventID:       eventID,
		TicketType:    domain.TicketVIP,
		Price:         100,
		TotalQuantity: 10,
	}
	payload, _ := json.Marshal(createBody)
	res, err := http.Post(srv.URL+"/inventory", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create inventory: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create inventory status %d: %s", res.StatusCode, b)
	}
	var inv domain.Inventory
	if err := json.NewDecoder(res.Body).Decode(&inv); err != nil {
		t.Fatalf("decode inventory: %v", err)
	}
	if inv.AvailableQuantity != 10 || inv.SoldQuantity != 0 {
		t.Fatalf("unexpected initial state: %+v", inv)
	}

	bookingID := fmt.Sprintf("book_%d", time.Now().UnixNano())
	holdPayload, _ := json.Marshal(domain.HoldRequest{
		EventID:    eventID,
		TicketType: domain.TicketVIP,
		Quantity:   2,
		BookingID:  bookingID,
	})
	res, err = http.Post(srv.URL+"/inventory/hold", "application/json", bytes.NewReader(holdPayload))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("hold status %d: %s", res.StatusCode, b)
	}

	confirmPayload, _ := json.Marshal(domain.BookingActionRequest{BookingID: bookingID})
	res, err = http.Post(srv.URL+"/inventory/confirm", "application/json", bytes.NewReader(confirmPayload))
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("confirm status %d: %s", res.StatusCode, b)
	}

	res, err = http.Get(srv.URL + "/inventory/event/" + eventID + "/availability")
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("availability status %d: %s", res.StatusCode, b)
	}
	var summary domain.AvailabilitySummary
	if err := json.NewDecoder(res.Body).Decode(&summary); err != nil {
		t.Fatalf("decode availability: %v", err)
	}
	if len(summary.Items) != 1 {
		t.Fatalf("expected 1 item, got %+v", summary.Items)
	}
	item := summary.Items[0]
	if item.SoldQuantity != 2 || item.AvailableQuantity != 8 || item.HeldQuantity != 0 {
		t.Fatalf("unexpected availability: %+v", item)
	}

	res, err = http.Get(srv.URL + "/inventory/" + inv.InventoryID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("get by id status %d: %s", res.StatusCode, b)
	}
	if err := json.NewDecoder(res.Body).Decode(&inv); err != nil {
		t.Fatalf("decode get by id: %v", err)
	}
	if inv.SoldQuantity != 2 || inv.AvailableQuantity != 8 {
		t.Fatalf("unexpected inventory after confirm: %+v", inv)
	}
}

func TestHoldInsufficientStock(t *testing.T) {
	srv := newTestServer(t)
	eventID := fmt.Sprintf("evt_soldout_%d", time.Now().UnixNano())

	createBody := domain.CreateInventoryRequest{
		EventID:       eventID,
		TicketType:    domain.TicketStandard,
		Price:         50,
		TotalQuantity: 1,
	}
	payload, _ := json.Marshal(createBody)
	res, err := http.Post(srv.URL+"/inventory", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", res.StatusCode)
	}

	holdPayload, _ := json.Marshal(domain.HoldRequest{
		EventID:    eventID,
		TicketType: domain.TicketStandard,
		Quantity:   2,
		BookingID:  fmt.Sprintf("book_fail_%d", time.Now().UnixNano()),
	})
	res, err = http.Post(srv.URL+"/inventory/hold", "application/json", bytes.NewReader(holdPayload))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 409 insufficient stock, got %d: %s", res.StatusCode, b)
	}
}
