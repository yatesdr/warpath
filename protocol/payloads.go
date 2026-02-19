package protocol

// --- Edge -> Core payloads ---

// EdgeRegister is sent by an edge on startup.
type EdgeRegister struct {
	NodeID   string   `json:"node_id"`
	Factory  string   `json:"factory"`
	Hostname string   `json:"hostname"`
	Version  string   `json:"version"`
	LineIDs  []string `json:"line_ids"`
}

// EdgeHeartbeat is sent periodically by an edge.
type EdgeHeartbeat struct {
	NodeID  string `json:"node_id"`
	Uptime  int64  `json:"uptime_s"`
	Orders  int    `json:"active_orders"`
}

// OrderRequest is a new transport order from edge.
type OrderRequest struct {
	OrderUUID       string  `json:"order_uuid"`
	OrderType       string  `json:"order_type"`
	PayloadTypeCode string  `json:"payload_type_code,omitempty"`
	PayloadDesc     string  `json:"payload_desc,omitempty"`
	Quantity        float64 `json:"quantity"`
	DeliveryNode    string  `json:"delivery_node,omitempty"`
	PickupNode      string  `json:"pickup_node,omitempty"`
	StagingNode     string  `json:"staging_node,omitempty"`
	LoadType        string  `json:"load_type,omitempty"`
	Priority        int     `json:"priority,omitempty"`
	RetrieveEmpty   bool    `json:"retrieve_empty,omitempty"`
}

// OrderCancel cancels an existing order.
type OrderCancel struct {
	OrderUUID string `json:"order_uuid"`
	Reason    string `json:"reason"`
}

// OrderReceipt confirms delivery acceptance.
type OrderReceipt struct {
	OrderUUID   string  `json:"order_uuid"`
	ReceiptType string  `json:"receipt_type"`
	FinalCount  float64 `json:"final_count"`
}

// OrderRedirect changes the delivery destination.
type OrderRedirect struct {
	OrderUUID       string `json:"order_uuid"`
	NewDeliveryNode string `json:"new_delivery_node"`
}

// OrderStorageWaybill submits a store order.
type OrderStorageWaybill struct {
	OrderUUID   string  `json:"order_uuid"`
	OrderType   string  `json:"order_type"`
	PayloadDesc string  `json:"payload_desc,omitempty"`
	PickupNode  string  `json:"pickup_node"`
	FinalCount  float64 `json:"final_count"`
}

// --- Core -> Edge payloads ---

// EdgeRegistered acknowledges edge registration.
type EdgeRegistered struct {
	NodeID  string `json:"node_id"`
	Message string `json:"message,omitempty"`
}

// EdgeHeartbeatAck acknowledges a heartbeat.
type EdgeHeartbeatAck struct {
	NodeID   string `json:"node_id"`
	ServerTS int64  `json:"server_ts"`
}

// OrderAck confirms order acceptance.
type OrderAck struct {
	OrderUUID     string `json:"order_uuid"`
	ShingoOrderID int64  `json:"shingo_order_id"`
	SourceNode    string `json:"source_node,omitempty"`
}

// OrderWaybill assigns a robot.
type OrderWaybill struct {
	OrderUUID string `json:"order_uuid"`
	WaybillID string `json:"waybill_id"`
	RobotID   string `json:"robot_id,omitempty"`
	ETA       string `json:"eta,omitempty"`
}

// OrderUpdate provides a status change.
type OrderUpdate struct {
	OrderUUID string `json:"order_uuid"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	ETA       string `json:"eta,omitempty"`
}

// OrderDelivered signals fleet delivery complete.
type OrderDelivered struct {
	OrderUUID   string `json:"order_uuid"`
	DeliveredAt string `json:"delivered_at"`
}

// OrderError signals order failure.
type OrderError struct {
	OrderUUID string `json:"order_uuid"`
	ErrorCode string `json:"error_code"`
	Detail    string `json:"detail"`
}

// OrderCancelled confirms order cancellation.
type OrderCancelled struct {
	OrderUUID string `json:"order_uuid"`
	Reason    string `json:"reason"`
}
