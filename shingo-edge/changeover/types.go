package changeover

// Changeover states
const (
	StateRunning     = "running"
	StateStopping    = "stopping"
	StateCountingOut = "counting_out"
	StateStoring     = "storing"
	StateDelivering  = "delivering"
	StateCountingIn  = "counting_in"
	StateReady       = "ready"
)

// stateOrder defines the linear progression of changeover states.
var stateOrder = []string{
	StateRunning,
	StateStopping,
	StateCountingOut,
	StateStoring,
	StateDelivering,
	StateCountingIn,
	StateReady,
	StateRunning, // loops back
}

// NextState returns the next state in the changeover sequence.
func NextState(current string) (string, bool) {
	for i, s := range stateOrder {
		if s == current && i < len(stateOrder)-1 {
			return stateOrder[i+1], true
		}
	}
	return "", false
}

// StateIndex returns the ordinal index of a state (for progress display).
func StateIndex(state string) int {
	for i, s := range stateOrder {
		if s == state {
			return i
		}
	}
	return -1
}

// AllStates returns the ordered list of changeover states (excluding terminal Running).
var AllStates = stateOrder[:len(stateOrder)-1]
