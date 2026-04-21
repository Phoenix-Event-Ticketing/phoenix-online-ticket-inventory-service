package handler

import (
	"net/http"
	"strings"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/auth"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter configures Gin with middleware and inventory routes.
// serviceName is returned by GET /inventory (public); empty uses "ticket-inventory-service".
func NewRouter(log *zap.Logger, inv *InventoryHandler, mw *auth.Middleware, serviceName string) *gin.Engine {
	if log == nil {
		log = zap.NewNop()
	}
	if mw == nil {
		mw = auth.NewMiddleware(nil)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	name := strings.TrimSpace(serviceName)
	if name == "" {
		name = "ticket-inventory-service"
	}
	r.GET("/inventory", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": name})
	})

	grp := r.Group("/inventory")
	grp.Use(mw.Authenticate())
	{
		grp.POST("", mw.RequirePermission(auth.CreateTicketType), inv.CreateInventory)
		grp.POST("/bulk", mw.RequirePermission(auth.CreateTicketType), inv.BulkCreate)
		grp.PUT("/:inventoryId", mw.RequirePermission(auth.UpdateTicketInventory), inv.Update)
		grp.GET("/event/:eventId", mw.RequirePermission(auth.ViewTicketInventory), inv.ListByEvent)
		grp.GET("/event/:eventId/availability", mw.RequirePermission(auth.ViewTicketInventory), inv.Availability)
		grp.GET("/:inventoryId", mw.RequirePermission(auth.ViewTicketInventory), inv.GetByID)
		grp.POST("/hold", mw.RequirePermission(auth.ReserveTicket), inv.Hold)
		grp.POST("/confirm", mw.RequirePermission(auth.ReserveTicket), inv.Confirm)
		grp.POST("/release", mw.RequirePermission(auth.ReserveTicket), inv.Release)
	}

	return r
}
