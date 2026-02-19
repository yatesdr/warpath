package messaging

import (
	"log"
	"sync"
	"time"

	"shingo/protocol"
	"shingocore/store"
)

// CoreHandler handles inbound protocol messages on the orders topic.
// It processes registration and heartbeat messages directly, and logs
// order messages (dispatcher wiring deferred to follow-up).
type CoreHandler struct {
	protocol.NoOpHandler

	db        *store.DB
	client    *Client
	factoryID string
	nodeID    string
	dispatchTopic string

	// Background goroutine for stale edge detection
	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewCoreHandler creates a handler for inbound edge messages.
func NewCoreHandler(db *store.DB, client *Client, factoryID, nodeID, dispatchTopic string) *CoreHandler {
	return &CoreHandler{
		db:            db,
		client:        client,
		factoryID:     factoryID,
		nodeID:        nodeID,
		dispatchTopic: dispatchTopic,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the stale-edge detection goroutine.
func (h *CoreHandler) Start() {
	go h.staleEdgeLoop()
}

// Stop halts the stale-edge detection goroutine.
func (h *CoreHandler) Stop() {
	h.stopOnce.Do(func() { close(h.stopCh) })
}

func (h *CoreHandler) HandleEdgeRegister(env *protocol.Envelope, p *protocol.EdgeRegister) {
	log.Printf("core_handler: edge registered: %s (factory=%s, hostname=%s, version=%s, lines=%v)",
		p.NodeID, p.Factory, p.Hostname, p.Version, p.LineIDs)

	if err := h.db.RegisterEdge(p.NodeID, p.Factory, p.Hostname, p.Version, p.LineIDs); err != nil {
		log.Printf("core_handler: register edge %s: %v", p.NodeID, err)
		return
	}

	// Send registration acknowledgement
	reply, err := protocol.NewReply(
		protocol.TypeEdgeRegistered,
		protocol.Address{Role: protocol.RoleCore, Node: h.nodeID, Factory: h.factoryID},
		protocol.Address{Role: protocol.RoleEdge, Node: p.NodeID, Factory: p.Factory},
		env.ID,
		&protocol.EdgeRegistered{NodeID: p.NodeID, Message: "registered"},
	)
	if err != nil {
		log.Printf("core_handler: build registered reply: %v", err)
		return
	}

	if err := h.client.PublishEnvelope(h.dispatchTopic, reply); err != nil {
		log.Printf("core_handler: publish registered reply: %v", err)
	}
}

func (h *CoreHandler) HandleEdgeHeartbeat(env *protocol.Envelope, p *protocol.EdgeHeartbeat) {
	if err := h.db.UpdateHeartbeat(p.NodeID); err != nil {
		log.Printf("core_handler: update heartbeat for %s: %v", p.NodeID, err)
		return
	}

	// Send heartbeat ack
	reply, err := protocol.NewReply(
		protocol.TypeEdgeHeartbeatAck,
		protocol.Address{Role: protocol.RoleCore, Node: h.nodeID, Factory: h.factoryID},
		protocol.Address{Role: protocol.RoleEdge, Node: p.NodeID, Factory: env.Src.Factory},
		env.ID,
		&protocol.EdgeHeartbeatAck{NodeID: p.NodeID, ServerTS: time.Now().Unix()},
	)
	if err != nil {
		log.Printf("core_handler: build heartbeat ack: %v", err)
		return
	}

	if err := h.client.PublishEnvelope(h.dispatchTopic, reply); err != nil {
		log.Printf("core_handler: publish heartbeat ack: %v", err)
	}
}

// Order message handlers log receipt during transition period.
// Full dispatcher wiring deferred to follow-up.

func (h *CoreHandler) HandleOrderRequest(env *protocol.Envelope, p *protocol.OrderRequest) {
	log.Printf("core_handler: order request from %s: uuid=%s type=%s", env.Src.Node, p.OrderUUID, p.OrderType)
}

func (h *CoreHandler) HandleOrderCancel(env *protocol.Envelope, p *protocol.OrderCancel) {
	log.Printf("core_handler: order cancel from %s: uuid=%s", env.Src.Node, p.OrderUUID)
}

func (h *CoreHandler) HandleOrderReceipt(env *protocol.Envelope, p *protocol.OrderReceipt) {
	log.Printf("core_handler: delivery receipt from %s: uuid=%s", env.Src.Node, p.OrderUUID)
}

func (h *CoreHandler) HandleOrderRedirect(env *protocol.Envelope, p *protocol.OrderRedirect) {
	log.Printf("core_handler: redirect from %s: uuid=%s -> %s", env.Src.Node, p.OrderUUID, p.NewDeliveryNode)
}

func (h *CoreHandler) HandleOrderStorageWaybill(env *protocol.Envelope, p *protocol.OrderStorageWaybill) {
	log.Printf("core_handler: storage waybill from %s: uuid=%s", env.Src.Node, p.OrderUUID)
}

func (h *CoreHandler) staleEdgeLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			n, err := h.db.MarkStaleEdges(180 * time.Second)
			if err != nil {
				log.Printf("core_handler: mark stale edges: %v", err)
			} else if n > 0 {
				log.Printf("core_handler: marked %d edge(s) stale", n)
			}
		}
	}
}
