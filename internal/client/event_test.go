package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewEventServiceClient_ValidatesInput(t *testing.T) {
	if _, err := NewEventServiceClient("   ", time.Second); err == nil {
		t.Fatal("expected error for empty base URL")
	}
	if _, err := NewEventServiceClient("::bad-url::", time.Second); err == nil {
		t.Fatal("expected error for invalid base URL")
	}
	c, err := NewEventServiceClient(" http://example.com/ ", 0)
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != "http://example.com" {
		t.Fatalf("unexpected base URL: %q", c.baseURL)
	}
	if c.httpClient.Timeout != 3*time.Second {
		t.Fatalf("unexpected default timeout: %s", c.httpClient.Timeout)
	}
}

func TestEventExists_StatusMappingAndPathEscape(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/events/evt-ok":
			w.WriteHeader(http.StatusOK)
		case "/events/evt-missing":
			w.WriteHeader(http.StatusNotFound)
		case "/events/evt-upstream":
			w.WriteHeader(http.StatusBadGateway)
		case "/events/evt with space":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c, err := NewEventServiceClient(ts.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := c.EventExists(context.Background(), "evt-ok")
	if err != nil || !ok {
		t.Fatalf("expected exists=true, got exists=%v err=%v", ok, err)
	}

	ok, err = c.EventExists(context.Background(), "evt-missing")
	if err != nil || ok {
		t.Fatalf("expected exists=false with nil error, got exists=%v err=%v", ok, err)
	}

	ok, err = c.EventExists(context.Background(), "evt-upstream")
	if err == nil || ok {
		t.Fatalf("expected upstream error, got exists=%v err=%v", ok, err)
	}

	ok, err = c.EventExists(context.Background(), "evt with space")
	if err != nil || !ok {
		t.Fatalf("expected escaped path to work, got exists=%v err=%v", ok, err)
	}
}

func TestEventExists_EmptyEventIDShortCircuits(t *testing.T) {
	c, err := NewEventServiceClient("http://example.com", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := c.EventExists(context.Background(), "   ")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Fatal("expected exists=false for empty event id")
	}
}

func TestEventExists_NetworkError(t *testing.T) {
	c, err := NewEventServiceClient("http://127.0.0.1:1", 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := c.EventExists(context.Background(), "evt-network")
	if err == nil || ok {
		t.Fatalf("expected network error, got exists=%v err=%v", ok, err)
	}
	if !strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "refused") {
		t.Fatalf("unexpected network error: %v", err)
	}
}
