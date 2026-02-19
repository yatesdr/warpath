package messaging

import (
	"log"

	"shingoedge/orders"
	"shingo/protocol"
)

// EdgeHandler handles inbound protocol messages on the dispatch topic.
// It delegates order reply messages to the orders.Manager.
type EdgeHandler struct {
	protocol.NoOpHandler

	orderMgr *orders.Manager
}

// NewEdgeHandler creates a handler for inbound core messages.
func NewEdgeHandler(orderMgr *orders.Manager) *EdgeHandler {
	return &EdgeHandler{orderMgr: orderMgr}
}

func (h *EdgeHandler) HandleEdgeRegistered(_ *protocol.Envelope, p *protocol.EdgeRegistered) {
	log.Printf("edge_handler: registration acknowledged: node=%s msg=%s", p.NodeID, p.Message)
}

func (h *EdgeHandler) HandleEdgeHeartbeatAck(_ *protocol.Envelope, p *protocol.EdgeHeartbeatAck) {
	log.Printf("edge_handler: heartbeat ack: node=%s server_ts=%d", p.NodeID, p.ServerTS)
}

func (h *EdgeHandler) HandleOrderAck(env *protocol.Envelope, p *protocol.OrderAck) {
	log.Printf("edge_handler: order ack: uuid=%s shingo_id=%d", p.OrderUUID, p.ShingoOrderID)
	if err := h.orderMgr.HandleDispatchReply(p.OrderUUID, "ack", "", "", p.SourceNode); err != nil {
		log.Printf("edge_handler: handle ack for %s: %v", p.OrderUUID, err)
	}
}

func (h *EdgeHandler) HandleOrderWaybill(env *protocol.Envelope, p *protocol.OrderWaybill) {
	log.Printf("edge_handler: order waybill: uuid=%s waybill=%s", p.OrderUUID, p.WaybillID)
	if err := h.orderMgr.HandleDispatchReply(p.OrderUUID, "waybill", p.WaybillID, p.ETA, ""); err != nil {
		log.Printf("edge_handler: handle waybill for %s: %v", p.OrderUUID, err)
	}
}

func (h *EdgeHandler) HandleOrderUpdate(env *protocol.Envelope, p *protocol.OrderUpdate) {
	log.Printf("edge_handler: order update: uuid=%s status=%s", p.OrderUUID, p.Status)
	if err := h.orderMgr.HandleDispatchReply(p.OrderUUID, "update", "", p.ETA, p.Detail); err != nil {
		log.Printf("edge_handler: handle update for %s: %v", p.OrderUUID, err)
	}
}

func (h *EdgeHandler) HandleOrderDelivered(env *protocol.Envelope, p *protocol.OrderDelivered) {
	log.Printf("edge_handler: order delivered: uuid=%s at=%s", p.OrderUUID, p.DeliveredAt)
	if err := h.orderMgr.HandleDispatchReply(p.OrderUUID, "delivered", "", "", p.DeliveredAt); err != nil {
		log.Printf("edge_handler: handle delivered for %s: %v", p.OrderUUID, err)
	}
}

func (h *EdgeHandler) HandleOrderError(env *protocol.Envelope, p *protocol.OrderError) {
	log.Printf("edge_handler: order error: uuid=%s code=%s detail=%s", p.OrderUUID, p.ErrorCode, p.Detail)
}

func (h *EdgeHandler) HandleOrderCancelled(env *protocol.Envelope, p *protocol.OrderCancelled) {
	log.Printf("edge_handler: order cancelled: uuid=%s reason=%s", p.OrderUUID, p.Reason)
}
