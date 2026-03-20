package handler

import (
	"errors"
	"net/http"

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

func notImplemented(c *gin.Context, err error) {
	if err != nil && errors.Is(err, service.ErrNotImplemented) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "not implemented",
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

// CreateInventory handles POST /inventory.
func (h *InventoryHandler) CreateInventory(c *gin.Context) {
	notImplemented(c, h.svc.CreateInventory(c.Request.Context()))
}

// BulkCreate handles POST /inventory/bulk.
func (h *InventoryHandler) BulkCreate(c *gin.Context) {
	notImplemented(c, h.svc.BulkCreate(c.Request.Context()))
}

// Update handles PUT /inventory/:inventoryId.
func (h *InventoryHandler) Update(c *gin.Context) {
	notImplemented(c, h.svc.Update(c.Request.Context()))
}

// ListByEvent handles GET /inventory/event/:eventId.
func (h *InventoryHandler) ListByEvent(c *gin.Context) {
	notImplemented(c, h.svc.ListByEvent(c.Request.Context()))
}

// Availability handles GET /inventory/event/:eventId/availability.
func (h *InventoryHandler) Availability(c *gin.Context) {
	notImplemented(c, h.svc.AvailabilitySummary(c.Request.Context()))
}

// GetByID handles GET /inventory/:inventoryId.
func (h *InventoryHandler) GetByID(c *gin.Context) {
	notImplemented(c, h.svc.GetByID(c.Request.Context()))
}

// Hold handles POST /inventory/hold.
func (h *InventoryHandler) Hold(c *gin.Context) {
	notImplemented(c, h.svc.Hold(c.Request.Context()))
}

// Confirm handles POST /inventory/confirm.
func (h *InventoryHandler) Confirm(c *gin.Context) {
	notImplemented(c, h.svc.Confirm(c.Request.Context()))
}

// Release handles POST /inventory/release.
func (h *InventoryHandler) Release(c *gin.Context) {
	notImplemented(c, h.svc.Release(c.Request.Context()))
}
