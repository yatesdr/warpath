package plc

// EventEmitter is the interface the PLC package uses to emit events.
// The engine package implements this via an adapter to avoid import cycles.
type EventEmitter interface {
	EmitCounterRead(rpID int64, plcName, tagName string, value int64)
	EmitCounterDelta(rpID, lineID, jobStyleID, delta, newCount int64)
	EmitCounterAnomaly(snapshotID, rpID int64, plcName, tagName string, oldVal, newVal int64, anomalyType string)
	EmitPLCConnected(plcName string)
	EmitPLCDisconnected(plcName string, err error)
	EmitWarLinkConnected()
	EmitWarLinkDisconnected(err error)
}
