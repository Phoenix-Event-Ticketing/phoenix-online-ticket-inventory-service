package repository

import (
	"strings"
	"testing"
)

func TestGenerateInventoryID(t *testing.T) {
	a := GenerateInventoryID()
	b := GenerateInventoryID()
	if a == b {
		t.Fatal("ids should differ")
	}
	if !strings.HasPrefix(a, "inv_") || !strings.HasPrefix(b, "inv_") {
		t.Fatalf("unexpected %q %q", a, b)
	}
}
