package orders

// Order types
const (
	TypeRetrieve = "retrieve"
	TypeStore    = "store"
	TypeMove     = "move"
)

// Order statuses
const (
	StatusQueued       = "queued"
	StatusSubmitted    = "submitted"
	StatusAcknowledged = "acknowledged"
	StatusInTransit    = "in_transit"
	StatusDelivered    = "delivered"
	StatusConfirmed    = "confirmed"
	StatusCancelled    = "cancelled"
)

// validTransitions defines which status transitions are allowed.
var validTransitions = map[string][]string{
	StatusQueued:       {StatusSubmitted, StatusCancelled},
	StatusSubmitted:    {StatusAcknowledged, StatusCancelled},
	StatusAcknowledged: {StatusInTransit, StatusCancelled},
	StatusInTransit:    {StatusDelivered, StatusCancelled},
	StatusDelivered:    {StatusConfirmed, StatusCancelled},
}

// IsValidTransition checks if a status transition is allowed.
func IsValidTransition(from, to string) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the status is a terminal state.
func IsTerminal(status string) bool {
	return status == StatusConfirmed || status == StatusCancelled
}
