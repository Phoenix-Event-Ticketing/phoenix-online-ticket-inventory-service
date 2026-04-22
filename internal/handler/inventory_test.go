package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/auth"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/config"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Environment: "test", AuthDisabled: "true"}
	svc := service.NewInventoryService(service.NewFakeInventoryRepo(), time.Minute)
	h := NewInventoryHandler(svc)
	return NewRouter(zap.NewNop(), h, auth.NewMiddleware(cfg), "ticket-inventory-service")
}

func TestRouter_Health(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRespondServiceErr_EventValidationMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "event not found", err: service.ErrEventNotFound, want: http.StatusNotFound},
		{name: "event service unavailable", err: service.ErrEventServiceUnavailable, want: http.StatusServiceUnavailable},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if !respondServiceErr(c, tc.err) {
				t.Fatal("expected handled error")
			}
			if w.Code != tc.want {
				t.Fatalf("want %d got %d", tc.want, w.Code)
			}
		})
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if !respondServiceErr(c, errors.New("unknown")) {
		t.Fatal("expected handled error")
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 got %d", w.Code)
	}
}

func TestRouter_GetInventoryRoot(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/inventory", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var body struct {
		Service string `json:"service"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Service != "ticket-inventory-service" {
		t.Fatalf("unexpected service %q", body.Service)
	}
}

func TestRouter_CreateInventory_InvalidJSON(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRouter_CreateInventory_Created(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	body := `{"eventId":"e1","ticketType":"VIP","price":10,"totalQuantity":10}`
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestRouter_GetByID_NotFound(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/inventory/does-not-exist", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRouter_Confirm_HoldNotFound(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory/confirm", strings.NewReader(`{"bookingId":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRouter_ListByEvent(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/inventory/event/e-empty", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRouter_HoldFlow(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"evt1","ticketType":"VIP","price":1,"totalQuantity":5}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory/hold", strings.NewReader(`{"eventId":"evt1","ticketType":"VIP","quantity":1,"bookingId":"b-flow"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hold %d %s", w.Code, w.Body.String())
	}
}

func TestRouter_CreateInventory_InvalidTicketType(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"e1","ticketType":"INVALID","price":1,"totalQuantity":1}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRouter_CreateInventory_DuplicateEventType(t *testing.T) {
	r := testRouter(t)
	body := `{"eventId":"e-dup","ticketType":"VIP","price":1,"totalQuantity":2}`
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("second %d", w.Code)
	}
}

func TestRouter_Hold_InsufficientStock(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"e-low","ticketType":"VIP","price":1,"totalQuantity":1}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory/hold", strings.NewReader(`{"eventId":"e-low","ticketType":"VIP","quantity":10,"bookingId":"b-low"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("hold %d", w.Code)
	}
}

func TestRouter_GetByID_AndUpdate(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"e-get","ticketType":"VIP","price":5,"totalQuantity":4}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d", w.Code)
	}
	var created map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["inventoryId"].(string)
	if id == "" {
		t.Fatal("no id")
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/inventory/"+id, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/inventory/"+id, strings.NewReader(`{"price":9.99}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("put %d %s", w.Code, w.Body.String())
	}
}

func TestRouter_ConfirmAndRelease(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"e-cr","ticketType":"STANDARD","price":2,"totalQuantity":6}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory/hold", strings.NewReader(`{"eventId":"e-cr","ticketType":"STANDARD","quantity":1,"bookingId":"b-cr"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hold %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory/confirm", strings.NewReader(`{"bookingId":"b-cr"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("confirm %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"e-rel","ticketType":"EARLY_BIRD","price":3,"totalQuantity":4}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create2 %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory/hold", strings.NewReader(`{"eventId":"e-rel","ticketType":"EARLY_BIRD","quantity":1,"bookingId":"b-rel"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hold2 %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/inventory/release", strings.NewReader(`{"bookingId":"b-rel"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("release %d %s", w.Code, w.Body.String())
	}
}

func TestRouter_EventAvailability(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader(`{"eventId":"e-av","ticketType":"VIP","price":1,"totalQuantity":3}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/inventory/event/e-av/availability", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("availability %d", w.Code)
	}
}

func TestRouter_InvalidJSONBodies(t *testing.T) {
	r := testRouter(t)
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/inventory/bulk"},
		{http.MethodPost, "/inventory/hold"},
		{http.MethodPut, "/inventory/some-id"},
		{http.MethodPost, "/inventory/confirm"},
		{http.MethodPost, "/inventory/release"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{`))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("want 400 got %d for %s %s", w.Code, tc.method, tc.path)
			}
		})
	}
}
