package protocol

// Message type constants for the unified protocol.
const (
	// Edge -> Core (published on orders topic)
	TypeEdgeRegister       = "edge.register"
	TypeEdgeHeartbeat      = "edge.heartbeat"
	TypeOrderRequest       = "order.request"
	TypeOrderCancel        = "order.cancel"
	TypeOrderReceipt       = "order.receipt"
	TypeOrderRedirect      = "order.redirect"
	TypeOrderStorageWaybill = "order.storage_waybill"

	// Core -> Edge (published on dispatch topic)
	TypeEdgeRegistered  = "edge.registered"
	TypeEdgeHeartbeatAck = "edge.heartbeat_ack"
	TypeOrderAck        = "order.ack"
	TypeOrderWaybill    = "order.waybill"
	TypeOrderUpdate     = "order.update"
	TypeOrderDelivered  = "order.delivered"
	TypeOrderError      = "order.error"
	TypeOrderCancelled  = "order.cancelled"
)

// Roles for Address.Role.
const (
	RoleEdge = "edge"
	RoleCore = "core"
)

// Protocol version.
const Version = 1
