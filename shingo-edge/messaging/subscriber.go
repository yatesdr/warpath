package messaging

import (
	"encoding/json"
	"log"

	"shingoedge/config"
	"shingoedge/orders"
)

// Subscriber listens for inbound dispatch replies and routes them to the order manager.
type Subscriber struct {
	client   *Client
	cfg      *config.Config
	orderMgr *orders.Manager
}

// NewSubscriber creates a new inbound message subscriber.
func NewSubscriber(client *Client, cfg *config.Config, orderMgr *orders.Manager) *Subscriber {
	return &Subscriber{
		client:   client,
		cfg:      cfg,
		orderMgr: orderMgr,
	}
}

// Start subscribes to the inbound topic and begins processing replies.
func (s *Subscriber) Start() error {
	return s.client.Subscribe(s.cfg.Messaging.InboundTopic, s.handleMessage)
}

func (s *Subscriber) handleMessage(payload []byte) {
	var reply DispatchReply
	if err := json.Unmarshal(payload, &reply); err != nil {
		log.Printf("unmarshal dispatch reply: %v", err)
		return
	}

	// Filter: only process messages for our namespace + line
	if reply.Namespace != s.cfg.Namespace || reply.LineID != s.cfg.LineID {
		return
	}

	if err := s.orderMgr.HandleDispatchReply(
		reply.OrderUUID,
		reply.ReplyType,
		reply.WaybillID,
		reply.ETA,
		reply.StatusDetail,
	); err != nil {
		log.Printf("handle dispatch reply for %s: %v", reply.OrderUUID, err)
	}
}
