package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/observability"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// InventoryHandler serves HTTP routes for ticket inventory.
type InventoryHandler struct {
	svc *service.InventoryService
}

// NewInventoryHandler constructs the handler.
func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

func writeError(c *gin.Context, status int, message string, code string, details interface{}) {
	traceID := ""
	requestID := ""
	if c.Request != nil {
		sc := trace.SpanFromContext(c.Request.Context()).SpanContext()
		if sc.IsValid() {
			traceID = sc.TraceID().String()
		}
		requestID = c.GetHeader("X-Request-Id")
	}

	body := gin.H{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"status":    status,
		"error":     http.StatusText(status),
		"errorCode": code,
		"message":   message,
		"requestId": requestID,
		"traceId":   traceID,
	}
	if details != nil {
		body["details"] = details
	}
	c.JSON(status, body)
}

func mapErrorCode(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidTicketType):
		return "VALIDATION_FAILED"
	case errors.Is(err, service.ErrNotFound):
		return "INVENTORY_NOT_FOUND"
	case errors.Is(err, service.ErrHoldNotFound):
		return "HOLD_NOT_FOUND"
	case errors.Is(err, service.ErrHoldExpired):
		return "HOLD_EXPIRED"
	case errors.Is(err, service.ErrDuplicateTicket):
		return "DUPLICATE_TICKET_TYPE"
	case errors.Is(err, service.ErrConflict):
		return "VALIDATION_FAILED"
	case errors.Is(err, service.ErrInsufficientStock):
		return "INSUFFICIENT_STOCK"
	case errors.Is(err, service.ErrHoldParamsMismatch):
		return "VALIDATION_FAILED"
	case errors.Is(err, service.ErrInvalidHoldState):
		return "VALIDATION_FAILED"
	case errors.Is(err, service.ErrEventNotFound):
		return "EVENT_NOT_FOUND"
	case errors.Is(err, service.ErrEventServiceUnavailable):
		return "EVENT_SERVICE_UNAVAILABLE"
	default:
		return "INTERNAL_ERROR"
	}
}

func respondServiceErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	code := mapErrorCode(err)
	switch {
	case errors.Is(err, domain.ErrInvalidTicketType):
		writeError(c, http.StatusBadRequest, "invalid ticketType", code, nil)
	case errors.Is(err, service.ErrNotFound):
		writeError(c, http.StatusNotFound, "not found", code, nil)
	case errors.Is(err, service.ErrHoldNotFound):
		writeError(c, http.StatusNotFound, "hold not found", code, nil)
	case errors.Is(err, service.ErrHoldExpired):
		writeError(c, http.StatusGone, "hold expired", code, nil)
	case errors.Is(err, service.ErrDuplicateTicket):
		writeError(c, http.StatusConflict, "duplicate ticket category for event", code, nil)
	case errors.Is(err, service.ErrConflict):
		writeError(c, http.StatusBadRequest, "invalid update", code, nil)
	case errors.Is(err, service.ErrInsufficientStock):
		observability.RecordStockConflict("hold")
		writeError(c, http.StatusConflict, "insufficient tickets available", code, nil)
	case errors.Is(err, service.ErrHoldParamsMismatch):
		writeError(c, http.StatusConflict, "hold parameters do not match existing hold", code, nil)
	case errors.Is(err, service.ErrInvalidHoldState):
		writeError(c, http.StatusConflict, "invalid hold state", code, nil)
	case errors.Is(err, service.ErrEventNotFound):
		observability.RecordEventValidationFailure("not_found")
		writeError(c, http.StatusNotFound, "event not found", code, nil)
	case errors.Is(err, service.ErrEventServiceUnavailable):
		observability.RecordEventValidationFailure("upstream_error")
		writeError(c, http.StatusServiceUnavailable, "event service unavailable", code, nil)
	default:
		writeError(c, http.StatusInternalServerError, "internal error", code, nil)
	}
	return true
}

// CreateInventory handles POST /inventory.
func (h *InventoryHandler) CreateInventory(c *gin.Context) {
	var req domain.CreateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body", "VALIDATION_FAILED", nil)
		return
	}
	inv, err := h.svc.CreateInventory(c.Request.Context(), req)
	if respondServiceErr(c, err) {
		return
	}
	c.JSON(http.StatusCreated, inv)
}

// BulkCreate handles POST /inventory/bulk.
func (h *InventoryHandler) BulkCreate(c *gin.Context) {
	var req domain.BulkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body", "VALIDATION_FAILED", nil)
		return
	}
	list, err := h.svc.BulkCreate(c.Request.Context(), req)
	if respondServiceErr(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"eventId": req.EventID, "items": list})
}

// Update handles PUT /inventory/:inventoryId.
func (h *InventoryHandler) Update(c *gin.Context) {
	inventoryID := c.Param("inventoryId")
	var req domain.UpdateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body", "VALIDATION_FAILED", nil)
		return
	}
	inv, err := h.svc.UpdateInventory(c.Request.Context(), inventoryID, req)
	if respondServiceErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, inv)
}

// ListByEvent handles GET /inventory/event/:eventId.
func (h *InventoryHandler) ListByEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	list, err := h.svc.ListByEvent(c.Request.Context(), eventID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"eventId": eventID, "items": list})
}

// Availability handles GET /inventory/event/:eventId/availability.
func (h *InventoryHandler) Availability(c *gin.Context) {
	eventID := c.Param("eventId")
	summary, err := h.svc.AvailabilitySummary(c.Request.Context(), eventID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetByID handles GET /inventory/:inventoryId.
func (h *InventoryHandler) GetByID(c *gin.Context) {
	inventoryID := c.Param("inventoryId")
	inv, err := h.svc.GetByID(c.Request.Context(), inventoryID)
	if respondServiceErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, inv)
}

// Hold handles POST /inventory/hold.
func (h *InventoryHandler) Hold(c *gin.Context) {
	var req domain.HoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body", "VALIDATION_FAILED", nil)
		return
	}
	resp, err := h.svc.Hold(c.Request.Context(), req, time.Now().UTC())
	if respondServiceErr(c, err) {
		observability.RecordHold(false, mapErrorCode(err))
		return
	}
	observability.RecordHold(true, "")
	c.JSON(http.StatusOK, resp)
}

// Confirm handles POST /inventory/confirm.
func (h *InventoryHandler) Confirm(c *gin.Context) {
	var req domain.BookingActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body", "VALIDATION_FAILED", nil)
		return
	}
	start := time.Now()
	hld, err := h.svc.Confirm(c.Request.Context(), req.BookingID, time.Now().UTC())
	observability.RecordConfirmDuration(time.Since(start))
	if respondServiceErr(c, err) {
		if errors.Is(err, service.ErrInsufficientStock) {
			observability.RecordStockConflict("confirm")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bookingId":  hld.BookingID,
		"holdStatus": hld.Status,
	})
}

// Release handles POST /inventory/release.
func (h *InventoryHandler) Release(c *gin.Context) {
	var req domain.BookingActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body", "VALIDATION_FAILED", nil)
		return
	}
	hld, err := h.svc.Release(c.Request.Context(), req.BookingID)
	if respondServiceErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bookingId":  hld.BookingID,
		"holdStatus": hld.Status,
	})
}
