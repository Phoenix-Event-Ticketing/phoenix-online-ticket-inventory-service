package service

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrDuplicateTicket    = errors.New("duplicate ticket type for event")
	ErrInsufficientStock  = errors.New("insufficient tickets available")
	ErrHoldNotFound       = errors.New("hold not found")
	ErrHoldExpired        = errors.New("hold expired")
	ErrInvalidHoldState   = errors.New("invalid hold state")
	ErrHoldParamsMismatch = errors.New("hold parameters do not match existing hold")
)
