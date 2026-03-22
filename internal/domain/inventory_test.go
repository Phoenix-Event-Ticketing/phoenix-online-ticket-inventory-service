package domain

import (
	"errors"
	"testing"
)

func TestValidTicketType(t *testing.T) {
	if err := ValidTicketType(TicketVIP); err != nil {
		t.Fatal(err)
	}
	if err := ValidTicketType(TicketType("")); err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(ValidTicketType(TicketType("X")), ErrInvalidTicketType) {
		t.Fatal("expected ErrInvalidTicketType")
	}
}
