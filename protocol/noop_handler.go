package protocol

// NoOpHandler implements MessageHandler with no-op methods.
// Embed this and override only the methods you need.
type NoOpHandler struct{}

func (NoOpHandler) HandleEdgeRegister(*Envelope, *EdgeRegister)             {}
func (NoOpHandler) HandleEdgeHeartbeat(*Envelope, *EdgeHeartbeat)           {}
func (NoOpHandler) HandleOrderRequest(*Envelope, *OrderRequest)             {}
func (NoOpHandler) HandleOrderCancel(*Envelope, *OrderCancel)               {}
func (NoOpHandler) HandleOrderReceipt(*Envelope, *OrderReceipt)             {}
func (NoOpHandler) HandleOrderRedirect(*Envelope, *OrderRedirect)           {}
func (NoOpHandler) HandleOrderStorageWaybill(*Envelope, *OrderStorageWaybill) {}
func (NoOpHandler) HandleEdgeRegistered(*Envelope, *EdgeRegistered)         {}
func (NoOpHandler) HandleEdgeHeartbeatAck(*Envelope, *EdgeHeartbeatAck)     {}
func (NoOpHandler) HandleOrderAck(*Envelope, *OrderAck)                     {}
func (NoOpHandler) HandleOrderWaybill(*Envelope, *OrderWaybill)             {}
func (NoOpHandler) HandleOrderUpdate(*Envelope, *OrderUpdate)               {}
func (NoOpHandler) HandleOrderDelivered(*Envelope, *OrderDelivered)         {}
func (NoOpHandler) HandleOrderError(*Envelope, *OrderError)                 {}
func (NoOpHandler) HandleOrderCancelled(*Envelope, *OrderCancelled)         {}

// Compile-time check that NoOpHandler implements MessageHandler.
var _ MessageHandler = NoOpHandler{}
