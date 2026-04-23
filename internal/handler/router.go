package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/auth"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/middleware"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/observability"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// NewRouter configures Gin with middleware and inventory routes.
// serviceName is returned by GET /inventory (public); empty uses "ticket-inventory-service".
func NewRouter(log *zap.Logger, inv *InventoryHandler, mw *auth.Middleware, serviceName string, metricsEnabled bool, corsAllowedOrigins []string) *gin.Engine {
	if log == nil {
		log = zap.NewNop()
	}
	if mw == nil {
		mw = auth.NewMiddleware(nil)
	}
	if len(corsAllowedOrigins) == 0 {
		corsAllowedOrigins = []string{"http://localhost:3000"}
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(otelgin.Middleware(serviceName))
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(log))
	r.Use(func(c *gin.Context) {
		c.Next()
		observability.RecordHTTPRequest(c.FullPath(), c.Request.Method, c.Writer.Status())
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if metricsEnabled {
		observability.Register()
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

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
