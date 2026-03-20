package domain

import "time"

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
	InventoryID       string     `json:"inventoryId" bson:"inventoryId"`
	EventID           string     `json:"eventId" bson:"eventId"`
	TicketType        TicketType `json:"ticketType" bson:"ticketType"`
	Price             float64    `json:"price" bson:"price"`
	TotalQuantity     int        `json:"totalQuantity" bson:"totalQuantity"`
	HeldQuantity      int        `json:"heldQuantity" bson:"heldQuantity"`
	SoldQuantity      int        `json:"soldQuantity" bson:"soldQuantity"`
	AvailableQuantity int        `json:"availableQuantity" bson:"availableQuantity"`
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
