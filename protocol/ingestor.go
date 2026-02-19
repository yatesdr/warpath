package protocol

import (
	"encoding/json"
	"log"
)

// FilterFunc returns true if the message should be processed.
type FilterFunc func(hdr *RawHeader) bool

// MessageHandler defines callbacks for all protocol message types.
// Embed NoOpHandler and override only the methods you need.
type MessageHandler interface {
	// Edge -> Core
	HandleEdgeRegister(env *Envelope, p *EdgeRegister)
	HandleEdgeHeartbeat(env *Envelope, p *EdgeHeartbeat)
	HandleOrderRequest(env *Envelope, p *OrderRequest)
	HandleOrderCancel(env *Envelope, p *OrderCancel)
	HandleOrderReceipt(env *Envelope, p *OrderReceipt)
	HandleOrderRedirect(env *Envelope, p *OrderRedirect)
	HandleOrderStorageWaybill(env *Envelope, p *OrderStorageWaybill)

	// Core -> Edge
	HandleEdgeRegistered(env *Envelope, p *EdgeRegistered)
	HandleEdgeHeartbeatAck(env *Envelope, p *EdgeHeartbeatAck)
	HandleOrderAck(env *Envelope, p *OrderAck)
	HandleOrderWaybill(env *Envelope, p *OrderWaybill)
	HandleOrderUpdate(env *Envelope, p *OrderUpdate)
	HandleOrderDelivered(env *Envelope, p *OrderDelivered)
	HandleOrderError(env *Envelope, p *OrderError)
	HandleOrderCancelled(env *Envelope, p *OrderCancelled)
}

// Ingestor performs two-phase decode and dispatches to a MessageHandler.
type Ingestor struct {
	handler MessageHandler
	filter  FilterFunc
}

// NewIngestor creates an ingestor with the given handler and filter.
func NewIngestor(handler MessageHandler, filter FilterFunc) *Ingestor {
	return &Ingestor{
		handler: handler,
		filter:  filter,
	}
}

// HandleRaw is the entry point for raw message bytes from the messaging layer.
func (ing *Ingestor) HandleRaw(data []byte) {
	// Phase 1: decode routing header only
	var hdr RawHeader
	if err := json.Unmarshal(data, &hdr); err != nil {
		log.Printf("protocol: header decode error: %v", err)
		return
	}

	// Check expiry
	if IsExpiredHeader(&hdr) {
		log.Printf("protocol: dropping expired message %s (type=%s)", hdr.ID, hdr.Type)
		return
	}

	// Apply filter
	if ing.filter != nil && !ing.filter(&hdr) {
		return
	}

	// Phase 2: full envelope decode
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		log.Printf("protocol: envelope decode error: %v", err)
		return
	}

	// Dispatch by type
	switch env.Type {
	case TypeEdgeRegister:
		decodeAndCall(ing.handler.HandleEdgeRegister, &env)
	case TypeEdgeHeartbeat:
		decodeAndCall(ing.handler.HandleEdgeHeartbeat, &env)
	case TypeOrderRequest:
		decodeAndCall(ing.handler.HandleOrderRequest, &env)
	case TypeOrderCancel:
		decodeAndCall(ing.handler.HandleOrderCancel, &env)
	case TypeOrderReceipt:
		decodeAndCall(ing.handler.HandleOrderReceipt, &env)
	case TypeOrderRedirect:
		decodeAndCall(ing.handler.HandleOrderRedirect, &env)
	case TypeOrderStorageWaybill:
		decodeAndCall(ing.handler.HandleOrderStorageWaybill, &env)
	case TypeEdgeRegistered:
		decodeAndCall(ing.handler.HandleEdgeRegistered, &env)
	case TypeEdgeHeartbeatAck:
		decodeAndCall(ing.handler.HandleEdgeHeartbeatAck, &env)
	case TypeOrderAck:
		decodeAndCall(ing.handler.HandleOrderAck, &env)
	case TypeOrderWaybill:
		decodeAndCall(ing.handler.HandleOrderWaybill, &env)
	case TypeOrderUpdate:
		decodeAndCall(ing.handler.HandleOrderUpdate, &env)
	case TypeOrderDelivered:
		decodeAndCall(ing.handler.HandleOrderDelivered, &env)
	case TypeOrderError:
		decodeAndCall(ing.handler.HandleOrderError, &env)
	case TypeOrderCancelled:
		decodeAndCall(ing.handler.HandleOrderCancelled, &env)
	default:
		log.Printf("protocol: unknown message type: %s", env.Type)
	}
}

// decodeAndCall unmarshals the payload and calls the handler method.
func decodeAndCall[T any](fn func(*Envelope, *T), env *Envelope) {
	var p T
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		log.Printf("protocol: payload decode error for %s: %v", env.Type, err)
		return
	}
	fn(env, &p)
}
