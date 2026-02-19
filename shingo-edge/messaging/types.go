package messaging

// DispatchReply is the inbound message from central dispatch.
type DispatchReply struct {
	Namespace    string `json:"namespace"`
	LineID       string `json:"line_id"`
	OrderUUID    string `json:"order_uuid"`
	ReplyType    string `json:"reply_type"` // ack, waybill, update, delivered
	WaybillID    string `json:"waybill_id,omitempty"`
	ETA          string `json:"eta,omitempty"`
	StatusDetail string `json:"status_detail,omitempty"`
	Timestamp    string `json:"timestamp"`
}
