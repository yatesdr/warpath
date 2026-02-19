package plc

// CalculateDelta computes the production delta between two counter readings.
// Returns the delta and any anomaly type ("reset", "jump", or "").
func CalculateDelta(lastCount, newCount, jumpThreshold int64) (delta int64, anomaly string) {
	if newCount == lastCount {
		return 0, ""
	}
	if newCount < lastCount {
		// Backward count = PLC restore/reset. Treat new count as production since reset.
		return newCount, "reset"
	}
	delta = newCount - lastCount
	if delta > jumpThreshold {
		return delta, "jump"
	}
	return delta, ""
}
