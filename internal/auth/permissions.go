package auth

// Permission names from Phoenix User Service spec §4.3 (Ticket Inventory Service).
const (
	ViewTicketInventory   = "VIEW_TICKET_INVENTORY"
	CreateTicketType      = "CREATE_TICKET_TYPE"
	UpdateTicketInventory = "UPDATE_TICKET_INVENTORY"
	DeleteTicketType      = "DELETE_TICKET_TYPE"
	ReserveTicket         = "RESERVE_TICKET"
)
