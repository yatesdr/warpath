package messaging

import (
	"log"
	"os"
	"sync"
	"time"

	"shingo/protocol"
)

// Heartbeater sends edge.register on startup and edge.heartbeat periodically.
type Heartbeater struct {
	client    *Client
	nodeID    string
	factory   string
	version   string
	lineIDs   []string
	topic     string // orders topic to publish on
	interval  time.Duration
	startTime time.Time

	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewHeartbeater creates a heartbeater for the given edge identity.
func NewHeartbeater(client *Client, nodeID, factory, version string, lineIDs []string, ordersTopic string) *Heartbeater {
	return &Heartbeater{
		client:   client,
		nodeID:   nodeID,
		factory:  factory,
		version:  version,
		lineIDs:  lineIDs,
		topic:    ordersTopic,
		interval: 60 * time.Second,
		stopCh:   make(chan struct{}),
	}
}

// Start sends an initial registration and begins the heartbeat loop.
func (h *Heartbeater) Start() {
	h.startTime = time.Now()
	h.sendRegister()
	go h.loop()
}

// Stop halts the heartbeat loop.
func (h *Heartbeater) Stop() {
	h.stopOnce.Do(func() { close(h.stopCh) })
}

func (h *Heartbeater) sendRegister() {
	hostname, _ := os.Hostname()
	env, err := protocol.NewEnvelope(
		protocol.TypeEdgeRegister,
		protocol.Address{Role: protocol.RoleEdge, Node: h.nodeID, Factory: h.factory},
		protocol.Address{Role: protocol.RoleCore},
		&protocol.EdgeRegister{
			NodeID:   h.nodeID,
			Factory:  h.factory,
			Hostname: hostname,
			Version:  h.version,
			LineIDs:  h.lineIDs,
		},
	)
	if err != nil {
		log.Printf("heartbeater: build register: %v", err)
		return
	}
	if err := h.client.PublishEnvelope(h.topic, env); err != nil {
		log.Printf("heartbeater: send register: %v", err)
	} else {
		log.Printf("heartbeater: sent edge.register (node=%s)", h.nodeID)
	}
}

func (h *Heartbeater) sendHeartbeat() {
	uptime := int64(time.Since(h.startTime).Seconds())
	env, err := protocol.NewEnvelope(
		protocol.TypeEdgeHeartbeat,
		protocol.Address{Role: protocol.RoleEdge, Node: h.nodeID, Factory: h.factory},
		protocol.Address{Role: protocol.RoleCore},
		&protocol.EdgeHeartbeat{
			NodeID: h.nodeID,
			Uptime: uptime,
		},
	)
	if err != nil {
		log.Printf("heartbeater: build heartbeat: %v", err)
		return
	}
	if err := h.client.PublishEnvelope(h.topic, env); err != nil {
		log.Printf("heartbeater: send heartbeat: %v", err)
	}
}

func (h *Heartbeater) loop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.sendHeartbeat()
		}
	}
}
