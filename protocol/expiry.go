package protocol

import "time"

// Default TTLs by message category.
var defaultTTLs = map[string]time.Duration{
	TypeEdgeHeartbeat:    90 * time.Second,
	TypeEdgeHeartbeatAck: 90 * time.Second,

	TypeEdgeRegister:   5 * time.Minute,
	TypeEdgeRegistered: 5 * time.Minute,

	TypeOrderRequest:        10 * time.Minute,
	TypeOrderCancel:         10 * time.Minute,
	TypeOrderRedirect:       10 * time.Minute,
	TypeOrderStorageWaybill: 10 * time.Minute,

	TypeOrderAck:    10 * time.Minute,
	TypeOrderUpdate: 10 * time.Minute,

	TypeOrderReceipt:   30 * time.Minute,
	TypeOrderWaybill:   30 * time.Minute,
	TypeOrderError:     30 * time.Minute,
	TypeOrderCancelled: 30 * time.Minute,

	TypeOrderDelivered: 60 * time.Minute,
}

// FallbackTTL is used when no specific TTL is configured.
const FallbackTTL = 10 * time.Minute

// DefaultTTLFor returns the default TTL for a message type.
func DefaultTTLFor(msgType string) time.Duration {
	if ttl, ok := defaultTTLs[msgType]; ok {
		return ttl
	}
	return FallbackTTL
}

// IsExpired returns true if the envelope has passed its expiry time.
func IsExpired(env *Envelope) bool {
	if env.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(env.ExpiresAt)
}

// IsExpiredHeader checks expiry using only the raw header.
func IsExpiredHeader(hdr *RawHeader) bool {
	if hdr.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(hdr.ExpiresAt)
}
