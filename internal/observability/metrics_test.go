package observability

import (
	"testing"
	"time"
)

func TestMetricsRecorders_DoNotPanic(t *testing.T) {
	Register()

	tests := []struct {
		name string
		fn   func()
	}{
		{name: "hold success", fn: func() { RecordHold(true, "") }},
		{name: "hold error default code", fn: func() { RecordHold(false, "") }},
		{name: "hold error custom code", fn: func() { RecordHold(false, "INSUFFICIENT_STOCK") }},
		{name: "event validation reason fallback", fn: func() { RecordEventValidationFailure("") }},
		{name: "event validation custom reason", fn: func() { RecordEventValidationFailure("not_found") }},
		{name: "stock conflict fallback", fn: func() { RecordStockConflict("") }},
		{name: "stock conflict operation", fn: func() { RecordStockConflict("confirm") }},
		{name: "confirm duration", fn: func() { RecordConfirmDuration(125 * time.Millisecond) }},
		{name: "http request fallback labels", fn: func() { RecordHTTPRequest("", "", 200) }},
		{name: "http request explicit labels", fn: func() { RecordHTTPRequest("/inventory", "GET", 200) }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.fn()
		})
	}
}

func TestRegister_Idempotent(t *testing.T) {
	Register()
	Register()
}
