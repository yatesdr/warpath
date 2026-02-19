package orders

// EventEmitter is the interface the orders package uses to emit events.
type EventEmitter interface {
	EmitOrderCreated(orderID int64, orderUUID, orderType string)
	EmitOrderStatusChanged(orderID int64, orderUUID, orderType, oldStatus, newStatus, eta string)
	EmitOrderCompleted(orderID int64, orderUUID, orderType string)
}
