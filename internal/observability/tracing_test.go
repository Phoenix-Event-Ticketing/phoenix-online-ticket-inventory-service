package observability

import (
	"context"
	"testing"
)

func TestInitTracing_DisabledWhenEndpointEmpty(t *testing.T) {
	shutdown, err := InitTracing(context.Background(), "inventory-service", "", "1.0")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected no-op shutdown, got %v", err)
	}
}

func TestInitTracing_InvalidSamplerFallsBack(t *testing.T) {
	shutdown, err := InitTracing(context.Background(), "inventory-service", "", "not-a-number")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function")
	}
}
