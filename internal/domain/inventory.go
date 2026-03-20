package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidTicketType is returned when ticketType is not VIP, STANDARD, or EARLY_BIRD.
var ErrInvalidTicketType = errors.New("invalid ticket type")

// TicketType identifies a ticket category for an event.
type TicketType string

const (
	TicketVIP       TicketType = "VIP"
	TicketStandard  TicketType = "STANDARD"
	TicketEarlyBird TicketType = "EARLY_BIRD"
)

// HoldStatus is the lifecycle state of a temporary hold (per outline).
type HoldStatus string

const (
	HoldHeld      HoldStatus = "HELD"
	HoldConfirmed HoldStatus = "CONFIRMED"
	HoldReleased  HoldStatus = "RELEASED"
)

// Inventory is a ticket inventory document stored in ticket_inventory.
type Inventory struct {
	InventoryID       string     `json:"inventoryId" bson:"_id"`
	EventID           string     `json:"eventId" bson:"eventId"`
	TicketType        TicketType `json:"ticketType" bson:"ticketType"`
	Price             float64    `json:"price" bson:"price"`
	TotalQuantity     int        `json:"totalQuantity" bson:"totalQuantity"`
	HeldQuantity      int        `json:"heldQuantity" bson:"heldQuantity"`
	SoldQuantity      int        `json:"soldQuantity" bson:"soldQuantity"`
	AvailableQuantity int        `json:"availableQuantity" bson:"availableQuantity"`
	CreatedAt         time.Time  `json:"createdAt,omitempty" bson:"createdAt,omitempty"`
	UpdatedAt         time.Time  `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
}

// CreateInventoryRequest is the body for POST /inventory.
type CreateInventoryRequest struct {
	EventID       string     `json:"eventId" binding:"required"`
	TicketType    TicketType `json:"ticketType" binding:"required"`
	Price         float64    `json:"price" binding:"gte=0"`
	TotalQuantity int        `json:"totalQuantity" binding:"required,min=1"`
}

// BulkCreateRequest is the body for POST /inventory/bulk.
type BulkCreateRequest struct {
	EventID string              `json:"eventId" binding:"required"`
	Items   []BulkInventoryItem `json:"items" binding:"required,min=1,dive"`
}

// BulkInventoryItem is one row in a bulk create.
type BulkInventoryItem struct {
	TicketType    TicketType `json:"ticketType" binding:"required"`
	Price         float64    `json:"price" binding:"gte=0"`
	TotalQuantity int        `json:"totalQuantity" binding:"required,min=1"`
}

// UpdateInventoryRequest is the body for PUT /inventory/{inventoryId}.
type UpdateInventoryRequest struct {
	TicketType    *TicketType `json:"ticketType"`
	Price         *float64    `json:"price" binding:"omitempty,gte=0"`
	TotalQuantity *int        `json:"totalQuantity" binding:"omitempty,min=0"`
}

// HoldRequest mirrors POST /inventory/hold body.
type HoldRequest struct {
	EventID    string     `json:"eventId" binding:"required"`
	TicketType TicketType `json:"ticketType" binding:"required"`
	Quantity   int        `json:"quantity" binding:"required,min=1"`
	BookingID  string     `json:"bookingId" binding:"required"`
}

// HoldResponse mirrors example hold response.
type HoldResponse struct {
	BookingID  string     `json:"bookingId"`
	HoldStatus HoldStatus `json:"holdStatus"`
	ExpiresAt  time.Time  `json:"expiresAt"`
}

// BookingActionRequest is the body for POST /inventory/confirm and /inventory/release.
type BookingActionRequest struct {
	BookingID string `json:"bookingId" binding:"required"`
}

// AvailabilityItem is one line in GET .../availability.
type AvailabilityItem struct {
	InventoryID       string     `json:"inventoryId"`
	TicketType        TicketType `json:"ticketType"`
	Price             float64    `json:"price"`
	TotalQuantity     int        `json:"totalQuantity"`
	HeldQuantity      int        `json:"heldQuantity"`
	SoldQuantity      int        `json:"soldQuantity"`
	AvailableQuantity int        `json:"availableQuantity"`
}

// AvailabilitySummary is the payload for GET /inventory/event/{eventId}/availability.
type AvailabilitySummary struct {
	EventID string             `json:"eventId"`
	Items   []AvailabilityItem `json:"items"`
}

// TicketHold is stored in ticket_holds; bookingId is the document _id.
type TicketHold struct {
	BookingID   string     `json:"bookingId" bson:"_id"`
	InventoryID string     `json:"inventoryId" bson:"inventoryId"`
	EventID     string     `json:"eventId" bson:"eventId"`
	TicketType  TicketType `json:"ticketType" bson:"ticketType"`
	Quantity    int        `json:"quantity" bson:"quantity"`
	Status      HoldStatus `json:"holdStatus" bson:"holdStatus"`
	ExpiresAt   time.Time  `json:"expiresAt" bson:"expiresAt"`
	CreatedAt   time.Time  `json:"createdAt,omitempty" bson:"createdAt"`
}

// ValidTicketType returns an error if t is not a supported ticket type.
func ValidTicketType(t TicketType) error {
	switch t {
	case TicketVIP, TicketStandard, TicketEarlyBird:
		return nil
	default:
		return fmt.Errorf("%w", ErrInvalidTicketType)
	}
}
