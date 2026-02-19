package seerrds

import "shingocore/fleet"

// emitterBridge adapts fleet.TrackerEmitter to rds.PollerEmitter.
type emitterBridge struct {
	emitter fleet.TrackerEmitter
}

func (b *emitterBridge) EmitOrderStatusChanged(orderID int64, rdsOrderID, oldStatus, newStatus, robotID, detail string) {
	b.emitter.EmitOrderStatusChanged(orderID, rdsOrderID, oldStatus, newStatus, robotID, detail)
}

// resolverBridge adapts fleet.OrderIDResolver to rds.OrderIDResolver.
type resolverBridge struct {
	resolver fleet.OrderIDResolver
}

func (b *resolverBridge) ResolveRDSOrderID(rdsOrderID string) (int64, error) {
	return b.resolver.ResolveVendorOrderID(rdsOrderID)
}
