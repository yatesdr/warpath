package fleet

// Backend is the vendor-neutral interface for fleet management systems.
// Implementations wrap vendor-specific APIs (Seer RDS, MiR, Locus, etc.).
type Backend interface {
	// CreateTransportOrder dispatches a transport order to the fleet backend.
	CreateTransportOrder(req TransportOrderRequest) (TransportOrderResult, error)

	// CancelOrder cancels a previously dispatched order.
	CancelOrder(vendorOrderID string) error

	// SetOrderPriority changes the priority of an active order.
	SetOrderPriority(vendorOrderID string, priority int) error

	// Ping checks connectivity to the fleet backend.
	Ping() error

	// Name returns a human-readable name for this backend (e.g. "SEER RDS").
	Name() string

	// MapState translates a vendor-specific state string to a dispatch status.
	MapState(vendorState string) string

	// IsTerminalState returns true if the vendor state represents a terminal state.
	IsTerminalState(vendorState string) bool

	// Reconfigure applies configuration changes at runtime.
	Reconfigure(cfg map[string]any)
}

// TransportOrderRequest contains vendor-neutral parameters for creating a transport order.
type TransportOrderRequest struct {
	OrderID    string // ShinGo-generated order ID (e.g. "sg-42-abc12345")
	ExternalID string // Edge UUID for correlation
	FromLoc    string // Source vendor location
	ToLoc      string // Destination vendor location
	Priority   int
}

// TransportOrderResult contains the result of a successful order creation.
type TransportOrderResult struct {
	VendorOrderID string // The ID assigned by the vendor system
}
