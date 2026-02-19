package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	src := Address{Role: RoleEdge, Node: "plant-a.line-1", Factory: "plant-a"}
	dst := Address{Role: RoleCore, Node: "", Factory: ""}

	env, err := NewEnvelope(TypeOrderRequest, src, dst, &OrderRequest{
		OrderUUID: "test-uuid-123",
		OrderType: "retrieve",
		Quantity:  10,
	})
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}

	if env.Version != Version {
		t.Errorf("version = %d, want %d", env.Version, Version)
	}
	if env.Type != TypeOrderRequest {
		t.Errorf("type = %q, want %q", env.Type, TypeOrderRequest)
	}
	if env.Src != src {
		t.Errorf("src = %+v, want %+v", env.Src, src)
	}
	if env.ID == "" {
		t.Error("ID should not be empty")
	}

	// Encode
	data, err := env.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Decode back
	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Type != TypeOrderRequest {
		t.Errorf("decoded type = %q, want %q", decoded.Type, TypeOrderRequest)
	}
	if decoded.ID != env.ID {
		t.Errorf("decoded id = %q, want %q", decoded.ID, env.ID)
	}

	// Decode payload
	var req OrderRequest
	if err := decoded.DecodePayload(&req); err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	if req.OrderUUID != "test-uuid-123" {
		t.Errorf("order_uuid = %q, want %q", req.OrderUUID, "test-uuid-123")
	}
	if req.Quantity != 10 {
		t.Errorf("quantity = %f, want 10", req.Quantity)
	}
}

func TestNewReply(t *testing.T) {
	reply, err := NewReply(TypeOrderAck,
		Address{Role: RoleCore},
		Address{Role: RoleEdge, Node: "plant-a.line-1"},
		"orig-msg-id",
		&OrderAck{OrderUUID: "uuid-1", ShingoOrderID: 42},
	)
	if err != nil {
		t.Fatalf("NewReply: %v", err)
	}
	if reply.CorID != "orig-msg-id" {
		t.Errorf("cor = %q, want %q", reply.CorID, "orig-msg-id")
	}
	if reply.Type != TypeOrderAck {
		t.Errorf("type = %q, want %q", reply.Type, TypeOrderAck)
	}
}

func TestExpiry(t *testing.T) {
	env := &Envelope{ExpiresAt: time.Now().UTC().Add(-1 * time.Minute)}
	if !IsExpired(env) {
		t.Error("expected expired envelope to be detected")
	}

	env.ExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	if IsExpired(env) {
		t.Error("expected future-expiry envelope to not be expired")
	}

	env.ExpiresAt = time.Time{}
	if IsExpired(env) {
		t.Error("expected zero-expiry envelope to not be expired")
	}
}

func TestExpiryHeader(t *testing.T) {
	hdr := &RawHeader{ExpiresAt: time.Now().UTC().Add(-1 * time.Second)}
	if !IsExpiredHeader(hdr) {
		t.Error("expected expired header to be detected")
	}

	hdr.ExpiresAt = time.Now().UTC().Add(5 * time.Minute)
	if IsExpiredHeader(hdr) {
		t.Error("expected future header to not be expired")
	}
}

func TestDefaultTTLFor(t *testing.T) {
	if ttl := DefaultTTLFor(TypeEdgeHeartbeat); ttl != 90*time.Second {
		t.Errorf("heartbeat TTL = %v, want 90s", ttl)
	}
	if ttl := DefaultTTLFor(TypeOrderDelivered); ttl != 60*time.Minute {
		t.Errorf("delivered TTL = %v, want 60m", ttl)
	}
	if ttl := DefaultTTLFor("unknown.type"); ttl != FallbackTTL {
		t.Errorf("unknown TTL = %v, want %v", ttl, FallbackTTL)
	}
}

func TestIngestorDispatch(t *testing.T) {
	handler := &testHandler{}
	ingestor := NewIngestor(handler, nil)

	// Build a valid envelope
	env, _ := NewEnvelope(TypeEdgeRegister,
		Address{Role: RoleEdge, Node: "test-node"},
		Address{Role: RoleCore},
		&EdgeRegister{NodeID: "test-node", Factory: "plant-a"},
	)
	data, _ := env.Encode()

	ingestor.HandleRaw(data)

	if !handler.registerCalled {
		t.Error("expected HandleEdgeRegister to be called")
	}
	if handler.registerPayload.NodeID != "test-node" {
		t.Errorf("node_id = %q, want %q", handler.registerPayload.NodeID, "test-node")
	}
}

func TestIngestorFilter(t *testing.T) {
	handler := &testHandler{}
	// Filter that rejects everything
	ingestor := NewIngestor(handler, func(_ *RawHeader) bool { return false })

	env, _ := NewEnvelope(TypeEdgeRegister,
		Address{Role: RoleEdge, Node: "test-node"},
		Address{Role: RoleCore},
		&EdgeRegister{NodeID: "test-node"},
	)
	data, _ := env.Encode()

	ingestor.HandleRaw(data)

	if handler.registerCalled {
		t.Error("expected handler to NOT be called when filter rejects")
	}
}

func TestIngestorDropsExpired(t *testing.T) {
	handler := &testHandler{}
	ingestor := NewIngestor(handler, nil)

	env, _ := NewEnvelope(TypeEdgeRegister,
		Address{Role: RoleEdge, Node: "test-node"},
		Address{Role: RoleCore},
		&EdgeRegister{NodeID: "test-node"},
	)
	// Force expiry in the past
	env.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	data, _ := env.Encode()

	ingestor.HandleRaw(data)

	if handler.registerCalled {
		t.Error("expected handler to NOT be called for expired message")
	}
}

func TestEdgeFilter(t *testing.T) {
	filter := func(hdr *RawHeader) bool {
		return hdr.Dst.Node == "plant-a.line-1" || hdr.Dst.Node == "*"
	}

	// Matching node
	if !filter(&RawHeader{Dst: Address{Node: "plant-a.line-1"}}) {
		t.Error("expected filter to accept matching node")
	}
	// Broadcast
	if !filter(&RawHeader{Dst: Address{Node: "*"}}) {
		t.Error("expected filter to accept broadcast")
	}
	// Other node
	if filter(&RawHeader{Dst: Address{Node: "plant-a.line-2"}}) {
		t.Error("expected filter to reject other node")
	}
}

func TestWireFormatKeys(t *testing.T) {
	env, _ := NewEnvelope(TypeEdgeHeartbeat,
		Address{Role: RoleEdge, Node: "n1", Factory: "f1"},
		Address{Role: RoleCore},
		&EdgeHeartbeat{NodeID: "n1", Uptime: 60},
	)
	data, _ := env.Encode()

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify short keys are used
	expected := []string{"v", "type", "id", "src", "dst", "ts", "exp", "p"}
	for _, k := range expected {
		if _, ok := m[k]; !ok {
			t.Errorf("expected key %q in wire format", k)
		}
	}
	// Verify long keys are NOT present
	long := []string{"version", "payload", "timestamp", "expires_at", "source", "destination"}
	for _, k := range long {
		if _, ok := m[k]; ok {
			t.Errorf("unexpected long key %q in wire format", k)
		}
	}
}

// testHandler tracks which methods were called.
type testHandler struct {
	NoOpHandler
	registerCalled  bool
	registerPayload EdgeRegister
}

func (h *testHandler) HandleEdgeRegister(env *Envelope, p *EdgeRegister) {
	h.registerCalled = true
	h.registerPayload = *p
}
