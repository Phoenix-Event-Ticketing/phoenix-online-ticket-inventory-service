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

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/auth"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/config"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/handler"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/repository"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/service"
	"github.com/golang-jwt/jwt/v5"
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

const integrationJWTSecret = "integration-test-jwt-secret"

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test (-short)")
	}

	t.Setenv("MONGODB_URI", mongoURI())
	t.Setenv("JWT_SECRET", integrationJWTSecret)
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("AUTH_DISABLED", "false")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
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
	authMW := auth.NewMiddleware(&cfg)
	router := handler.NewRouter(zap.NewNop(), invHandler, authMW, cfg.ServiceName)
	srv := httptest.NewServer(router)

	t.Cleanup(func() {
		srv.Close()
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return srv
}

func bearerUserToken(t *testing.T) string {
	t.Helper()
	claims := &auth.Claims{
		Sub: "integration-user",
		Permissions: []string{
			auth.CreateTicketType,
			auth.UpdateTicketInventory,
			auth.ViewTicketInventory,
			auth.ReserveTicket,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	s, err := auth.SignHS256(integrationJWTSecret, claims)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + s
}

func bearerServiceToken(t *testing.T, serviceID string) string {
	t.Helper()
	claims := &auth.Claims{
		Sub: serviceID,
		Typ: "service",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	s, err := auth.SignHS256(integrationJWTSecret, claims)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + s
}

func postJSON(t *testing.T, srv *httptest.Server, path, authHeader string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func getWithAuth(t *testing.T, srv *httptest.Server, path, authHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
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
	res := postJSON(t, srv, "/inventory", bearerUserToken(t), payload)
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
	res = postJSON(t, srv, "/inventory/hold", bearerUserToken(t), holdPayload)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("hold status %d: %s", res.StatusCode, b)
	}

	confirmPayload, _ := json.Marshal(domain.BookingActionRequest{BookingID: bookingID})
	res = postJSON(t, srv, "/inventory/confirm", bearerUserToken(t), confirmPayload)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("confirm status %d: %s", res.StatusCode, b)
	}

	res = getWithAuth(t, srv, "/inventory/event/"+eventID+"/availability", bearerUserToken(t))
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

	res = getWithAuth(t, srv, "/inventory/"+inv.InventoryID, bearerUserToken(t))
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
	res := postJSON(t, srv, "/inventory", bearerUserToken(t), payload)
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
	res = postJSON(t, srv, "/inventory/hold", bearerUserToken(t), holdPayload)
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 409 insufficient stock, got %d: %s", res.StatusCode, b)
	}
}

func TestServiceRegistryReserve(t *testing.T) {
	t.Setenv("SERVICE_REGISTRY", `{"booking-service":["RESERVE_TICKET"]}`)
	srv := newTestServer(t)
	eventID := fmt.Sprintf("evt_svc_%d", time.Now().UnixNano())

	createBody := domain.CreateInventoryRequest{
		EventID:       eventID,
		TicketType:    domain.TicketVIP,
		Price:         100,
		TotalQuantity: 5,
	}
	payload, _ := json.Marshal(createBody)
	res := postJSON(t, srv, "/inventory", bearerUserToken(t), payload)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create inventory status %d: %s", res.StatusCode, b)
	}

	bookingID := fmt.Sprintf("book_svc_%d", time.Now().UnixNano())
	holdPayload, _ := json.Marshal(domain.HoldRequest{
		EventID:    eventID,
		TicketType: domain.TicketVIP,
		Quantity:   1,
		BookingID:  bookingID,
	})
	res = postJSON(t, srv, "/inventory/hold", bearerServiceToken(t, "booking-service"), holdPayload)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("hold with service token status %d: %s", res.StatusCode, b)
	}

	res = getWithAuth(t, srv, "/inventory/event/"+eventID, bearerServiceToken(t, "booking-service"))
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 403 for VIEW without registry entry, got %d: %s", res.StatusCode, b)
	}
}
