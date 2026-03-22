package handler

import (
	"net/http"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter configures Gin with middleware and inventory routes.
func NewRouter(log *zap.Logger, inv *InventoryHandler) *gin.Engine {
	if log == nil {
		log = zap.NewNop()
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	grp := r.Group("/inventory")
	{
		grp.POST("", inv.CreateInventory)
		grp.POST("/bulk", inv.BulkCreate)
		grp.PUT("/:inventoryId", inv.Update)
		grp.GET("/event/:eventId", inv.ListByEvent)
		grp.GET("/event/:eventId/availability", inv.Availability)
		grp.GET("/:inventoryId", inv.GetByID)
		grp.POST("/hold", inv.Hold)
		grp.POST("/confirm", inv.Confirm)
		grp.POST("/release", inv.Release)
	}

	return r
}
