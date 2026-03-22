package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	log, err := New("info", "svc", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = log.Sync()
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New("not-a-real-level", "svc", "test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseLevel_Empty(t *testing.T) {
	lvl, err := parseLevel("")
	if err != nil {
		t.Fatal(err)
	}
	if lvl.String() != "info" {
		t.Fatal(lvl)
	}
}
