package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/domain"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/service"
	"github.com/gin-gonic/gin"
)

// InventoryHandler serves HTTP routes for ticket inventory.
type InventoryHandler struct {
	svc *service.InventoryService
}

// NewInventoryHandler constructs the handler.
func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

func respondServiceErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, domain.ErrInvalidTicketType):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticketType"})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, service.ErrHoldNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "hold not found"})
	case errors.Is(err, service.ErrHoldExpired):
		c.JSON(http.StatusGone, gin.H{"error": "hold expired"})
	case errors.Is(err, service.ErrDuplicateTicket):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate ticket category for event"})
	case errors.Is(err, service.ErrConflict):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid update"})
	case errors.Is(err, service.ErrInsufficientStock):
		c.JSON(http.StatusConflict, gin.H{"error": "insufficient tickets available"})
	case errors.Is(err, service.ErrHoldParamsMismatch):
		c.JSON(http.StatusConflict, gin.H{"error": "hold parameters do not match existing hold"})
	case errors.Is(err, service.ErrInvalidHoldState):
		c.JSON(http.StatusConflict, gin.H{"error": "invalid hold state"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
	return true
}

// CreateInventory handles POST /inventory.
func (h *InventoryHandler) CreateInventory(c *gin.Context) {
	var req domain.CreateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"eventId": eventID, "items": list})
}

// Availability handles GET /inventory/event/:eventId/availability.
func (h *InventoryHandler) Availability(c *gin.Context) {
	eventID := c.Param("eventId")
	summary, err := h.svc.AvailabilitySummary(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	resp, err := h.svc.Hold(c.Request.Context(), req, time.Now().UTC())
	if respondServiceErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Confirm handles POST /inventory/confirm.
func (h *InventoryHandler) Confirm(c *gin.Context) {
	var req domain.BookingActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	hld, err := h.svc.Confirm(c.Request.Context(), req.BookingID, time.Now().UTC())
	if respondServiceErr(c, err) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
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
