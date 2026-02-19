package changeover

import (
	"fmt"
	"log"
	"sync"

	"shingoedge/store"
)

// Machine manages the changeover state machine for a production line.
type Machine struct {
	mu           sync.Mutex
	db           *store.DB
	emitter      EventEmitter
	lineID       int64
	lineName     string
	fromJobStyle string
	toJobStyle   string
	state        string
	active       bool
}

// NewMachine creates a changeover state machine for a specific production line.
func NewMachine(db *store.DB, emitter EventEmitter, lineID int64, lineName string) *Machine {
	return &Machine{
		db:       db,
		emitter:  emitter,
		lineID:   lineID,
		lineName: lineName,
		state:    StateRunning,
	}
}

// LineID returns the production line ID for this machine.
func (m *Machine) LineID() int64 {
	return m.lineID
}

// LineName returns the production line name for this machine.
func (m *Machine) LineName() string {
	return m.lineName
}

// IsActive returns whether a changeover is in progress.
func (m *Machine) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}

// CurrentState returns the current changeover state.
func (m *Machine) CurrentState() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Info returns the current changeover info.
func (m *Machine) Info() (fromJobStyle, toJobStyle, state string, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fromJobStyle, m.toJobStyle, m.state, m.active
}

// Start begins a changeover from one job style to another.
func (m *Machine) Start(fromJobStyle, toJobStyle, operator string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.active {
		return fmt.Errorf("changeover already in progress from %s to %s", m.fromJobStyle, m.toJobStyle)
	}

	m.fromJobStyle = fromJobStyle
	m.toJobStyle = toJobStyle
	m.state = StateStopping
	m.active = true

	m.logTransition(StateRunning, StateStopping, "changeover initiated", operator)
	m.emitter.EmitChangeoverStarted(m.lineID, fromJobStyle, toJobStyle)
	m.emitter.EmitChangeoverStateChanged(m.lineID, fromJobStyle, toJobStyle, StateRunning, StateStopping)

	return nil
}

// Advance moves to the next state in the changeover sequence.
func (m *Machine) Advance(operator string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.active {
		return fmt.Errorf("no changeover in progress")
	}

	next, ok := NextState(m.state)
	if !ok {
		return fmt.Errorf("no next state from %s", m.state)
	}

	oldState := m.state
	m.state = next

	m.logTransition(oldState, next, "", operator)
	m.emitter.EmitChangeoverStateChanged(m.lineID, m.fromJobStyle, m.toJobStyle, oldState, next)

	// If we've returned to Running, the changeover is complete
	if next == StateRunning {
		m.active = false
		m.emitter.EmitChangeoverCompleted(m.lineID, m.fromJobStyle, m.toJobStyle)
	}

	return nil
}

// Cancel aborts the current changeover and resets to running.
func (m *Machine) Cancel() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.active {
		return fmt.Errorf("no changeover in progress")
	}

	oldState := m.state
	m.state = StateRunning
	m.active = false

	m.logTransition(oldState, StateRunning, "changeover cancelled", "")
	m.emitter.EmitChangeoverStateChanged(m.lineID, m.fromJobStyle, m.toJobStyle, oldState, StateRunning)
	m.emitter.EmitChangeoverCompleted(m.lineID, m.fromJobStyle, m.toJobStyle)

	return nil
}

func (m *Machine) logTransition(oldState, newState, detail, operator string) {
	if _, err := m.db.InsertChangeoverLog(m.fromJobStyle, m.toJobStyle, newState, detail, operator, m.lineID); err != nil {
		log.Printf("insert changeover log: %v", err)
	}
}
