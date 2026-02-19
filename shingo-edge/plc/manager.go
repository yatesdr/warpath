package plc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"shingoedge/config"
	"shingoedge/store"
)

// --- WarLink API response types ---

type warlinkPLCResponse struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	Slot        int    `json:"slot"`
	Status      string `json:"status"`
	ProductName string `json:"product_name"`
	Error       string `json:"error"`
}

type warlinkTagResponse struct {
	PLC   string      `json:"plc"`
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
	Error string      `json:"error"`
}

// TagInfo describes a tag available on a PLC.
type TagInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TagValue is a cached tag value from WarLink.
type TagValue struct {
	Name    string
	TypeStr string
	Value   interface{}
	Error   string
}

// ManagedPLC tracks per-PLC state from WarLink discovery.
type ManagedPLC struct {
	Name   string
	Status string
	Error  string
	Values map[string]TagValue
	mu     sync.RWMutex
}

// Manager manages PLC data via WarLink HTTP polling and counter polling.
type Manager struct {
	mu      sync.RWMutex
	db      *store.DB
	cfg     *config.Config
	emitter EventEmitter
	client  http.Client
	plcs    map[string]*ManagedPLC

	warlinkConnected bool
	warlinkError     error

	stopChan        chan struct{}
	warlinkStopChan chan struct{}
	warlinkRunning  bool
	warlinkWg       sync.WaitGroup
	wg              sync.WaitGroup
}

// NewManager creates a PLC manager.
func NewManager(db *store.DB, cfg *config.Config, emitter EventEmitter) *Manager {
	return &Manager{
		db:      db,
		cfg:     cfg,
		emitter: emitter,
		client:  http.Client{Timeout: 10 * time.Second},
		plcs:    make(map[string]*ManagedPLC),

		stopChan: make(chan struct{}),
	}
}

// baseURL returns the current WarLink base URL from config, trimming trailing slashes.
func (m *Manager) baseURL() string {
	return strings.TrimRight(m.cfg.WarLink.URL, "/")
}

// StartWarLinkPoller starts the goroutine that polls WarLink for PLC and tag data.
func (m *Manager) StartWarLinkPoller() {
	m.mu.Lock()
	if m.warlinkRunning {
		m.mu.Unlock()
		return
	}
	m.warlinkStopChan = make(chan struct{})
	m.warlinkRunning = true
	m.mu.Unlock()

	m.warlinkWg.Add(1)
	go m.warlinkPollLoop()
}

// StopWarLinkPoller stops the WarLink polling goroutine and resets connection state.
func (m *Manager) StopWarLinkPoller() {
	m.mu.Lock()
	if !m.warlinkRunning {
		m.mu.Unlock()
		return
	}
	close(m.warlinkStopChan)
	m.warlinkRunning = false
	m.mu.Unlock()

	m.warlinkWg.Wait()

	// Reset connection state
	if m.warlinkConnected {
		m.warlinkConnected = false
		m.warlinkError = nil
		m.emitter.EmitWarLinkDisconnected(nil)
	}

	// Mark all tracked PLCs as disconnected
	m.mu.Lock()
	for _, mp := range m.plcs {
		if mp.Status == "Connected" {
			mp.Status = "Disconnected"
			m.emitter.EmitPLCDisconnected(mp.Name, nil)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) warlinkPollLoop() {
	defer m.warlinkWg.Done()

	getPollRate := func() time.Duration {
		d := m.cfg.WarLink.PollRate
		if d <= 0 {
			return 2 * time.Second
		}
		return d
	}

	ticker := time.NewTicker(getPollRate())
	defer ticker.Stop()

	// Do an immediate first poll
	m.warlinkPollTick()

	for {
		select {
		case <-m.stopChan:
			return
		case <-m.warlinkStopChan:
			return
		case <-ticker.C:
			m.warlinkPollTick()
			ticker.Reset(getPollRate())
		}
	}
}

func (m *Manager) warlinkPollTick() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	plcs, err := m.fetchPLCs(ctx)
	if err != nil {
		if m.warlinkConnected {
			m.warlinkConnected = false
			m.warlinkError = err
			log.Printf("WarLink connection lost: %v", err)
			m.emitter.EmitWarLinkDisconnected(err)
		}
		return
	}

	wasDisconnected := !m.warlinkConnected
	if wasDisconnected {
		m.warlinkConnected = true
		m.warlinkError = nil
		log.Printf("WarLink connected: %s", m.baseURL())
	}

	// Track which PLCs we've seen this tick for status transitions
	seen := make(map[string]bool)

	for _, p := range plcs {
		seen[p.Name] = true

		m.mu.RLock()
		existing, exists := m.plcs[p.Name]
		m.mu.RUnlock()

		if !exists {
			existing = &ManagedPLC{
				Name:   p.Name,
				Values: make(map[string]TagValue),
			}
			m.mu.Lock()
			m.plcs[p.Name] = existing
			m.mu.Unlock()
		}

		oldStatus := existing.Status
		existing.Status = p.Status
		existing.Error = p.Error

		// Emit connection transitions
		if p.Status == "Connected" && oldStatus != "Connected" {
			m.emitter.EmitPLCConnected(p.Name)
		} else if p.Status != "Connected" && oldStatus == "Connected" {
			var emitErr error
			if p.Error != "" {
				emitErr = fmt.Errorf("%s", p.Error)
			}
			m.emitter.EmitPLCDisconnected(p.Name, emitErr)
		}

		// Fetch tags for connected PLCs
		if p.Status == "Connected" {
			tags, err := m.fetchTags(ctx, p.Name)
			if err != nil {
				log.Printf("WarLink fetch tags %s: %v", p.Name, err)
				continue
			}
			m.applyTags(p.Name, tags)
		}
	}

	// Detect PLCs that disappeared from WarLink
	m.mu.RLock()
	var removed []string
	for name, mp := range m.plcs {
		if !seen[name] && mp.Status == "Connected" {
			removed = append(removed, name)
		}
	}
	m.mu.RUnlock()

	for _, name := range removed {
		m.mu.RLock()
		mp := m.plcs[name]
		m.mu.RUnlock()
		mp.Status = "Disconnected"
		m.emitter.EmitPLCDisconnected(name, fmt.Errorf("removed from WarLink"))
	}

	// Emit after all PLCs are in the map so the API returns the full list
	if wasDisconnected {
		m.emitter.EmitWarLinkConnected()
	}
}

func (m *Manager) fetchPLCs(ctx context.Context) ([]warlinkPLCResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.baseURL()+"/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WarLink returned %d", resp.StatusCode)
	}
	var plcs []warlinkPLCResponse
	if err := json.NewDecoder(resp.Body).Decode(&plcs); err != nil {
		return nil, fmt.Errorf("decode PLCs: %w", err)
	}
	return plcs, nil
}

func (m *Manager) fetchTags(ctx context.Context, plcName string) (map[string]warlinkTagResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.baseURL()+"/"+plcName+"/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WarLink tags %s returned %d", plcName, resp.StatusCode)
	}
	var tags map[string]warlinkTagResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode tags %s: %w", plcName, err)
	}
	return tags, nil
}

func (m *Manager) applyTags(plcName string, tags map[string]warlinkTagResponse) {
	m.mu.RLock()
	mp, ok := m.plcs[plcName]
	m.mu.RUnlock()
	if !ok {
		return
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Build fresh values map, stripping PLCName. prefix from keys
	prefix := plcName + "."
	newValues := make(map[string]TagValue, len(tags))
	for key, tag := range tags {
		name := strings.TrimPrefix(key, prefix)
		newValues[name] = TagValue{
			Name:    name,
			TypeStr: tag.Type,
			Value:   tag.Value,
			Error:   tag.Error,
		}
	}
	mp.Values = newValues
}

// IsWarLinkConnected returns whether we can reach WarLink.
func (m *Manager) IsWarLinkConnected() bool {
	return m.warlinkConnected
}

// WarLinkError returns the last WarLink connection error, if any.
func (m *Manager) WarLinkError() error {
	return m.warlinkError
}

// PLCNames returns the names of all discovered PLCs, sorted.
func (m *Manager) PLCNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.plcs))
	for name := range m.plcs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetPLC returns the managed PLC state, or nil if not found.
func (m *Manager) GetPLC(name string) *ManagedPLC {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plcs[name]
}

// PLCStatuses returns a map of PLC name to connection status for all discovered PLCs.
func (m *Manager) PLCStatuses() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make(map[string]string, len(m.plcs))
	for name, mp := range m.plcs {
		statuses[name] = mp.Status
	}
	return statuses
}

// IsConnected returns whether a PLC is currently connected via WarLink.
func (m *Manager) IsConnected(name string) bool {
	m.mu.RLock()
	mp, ok := m.plcs[name]
	m.mu.RUnlock()
	return ok && mp.Status == "Connected"
}

// ReadTag reads a single tag from the WarLink cache.
func (m *Manager) ReadTag(plcName, tagName string) (interface{}, error) {
	m.mu.RLock()
	mp, ok := m.plcs[plcName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("PLC %s not found", plcName)
	}

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	tv, ok := mp.Values[tagName]
	if !ok {
		return nil, fmt.Errorf("tag %s not found on %s", tagName, plcName)
	}
	if tv.Error != "" {
		return nil, fmt.Errorf("tag %s error: %s", tagName, tv.Error)
	}
	return tv.Value, nil
}

// DiscoverTags returns all tags from the WarLink cache for a PLC.
func (m *Manager) DiscoverTags(plcName string) ([]TagInfo, error) {
	m.mu.RLock()
	mp, ok := m.plcs[plcName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("PLC %s not found", plcName)
	}

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	tags := make([]TagInfo, 0, len(mp.Values))
	for _, tv := range mp.Values {
		tags = append(tags, TagInfo{Name: tv.Name, Type: tv.TypeStr})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	return tags, nil
}

// StartPolling starts the counter polling loop for all enabled reporting points.
func (m *Manager) StartPolling() {
	m.wg.Add(1)
	go m.pollLoop()
}

// Stop stops the polling loops and cleans up.
func (m *Manager) Stop() {
	select {
	case <-m.stopChan:
	default:
		close(m.stopChan)
	}
	m.wg.Wait()
}

// --- Polling ---

func (m *Manager) pollLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.cfg.PollRate)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.pollAllReportingPoints()
		}
	}
}

func (m *Manager) pollAllReportingPoints() {
	rps, err := m.db.ListEnabledReportingPoints()
	if err != nil {
		log.Printf("list reporting points: %v", err)
		return
	}

	for _, rp := range rps {
		m.pollReportingPoint(rp)
	}
}

func (m *Manager) pollReportingPoint(rp store.ReportingPoint) {
	val, err := m.ReadTag(rp.PLCName, rp.TagName)
	if err != nil {
		return
	}

	newCount, ok := toInt64(val)
	if !ok {
		return
	}

	m.emitter.EmitCounterRead(rp.ID, rp.PLCName, rp.TagName, newCount)

	delta, anomaly := CalculateDelta(rp.LastCount, newCount, m.cfg.Counter.JumpThreshold)
	if delta == 0 && anomaly == "" {
		return
	}

	// Record snapshot
	confirmed := anomaly != "jump"
	snapID, err := m.db.InsertCounterSnapshot(rp.ID, newCount, delta, anomaly, confirmed)
	if err != nil {
		log.Printf("insert counter snapshot: %v", err)
		return
	}

	// Update the reporting point's last known count
	if err := m.db.UpdateReportingPointCounter(rp.ID, newCount); err != nil {
		log.Printf("update reporting point counter: %v", err)
	}

	if anomaly != "" {
		m.emitter.EmitCounterAnomaly(snapID, rp.ID, rp.PLCName, rp.TagName, rp.LastCount, newCount, anomaly)
	}

	// Resolve effective job style ID and line ID
	var lineID int64
	if rp.LineID != nil {
		lineID = *rp.LineID
	}
	effectiveJSID := rp.JobStyleID
	if rp.LineID != nil && rp.JobStyleID == 0 {
		// Use line's active style
		jsID, err := m.db.GetActiveJobStyleID(*rp.LineID)
		if err != nil || jsID == nil {
			return // no active style
		}
		effectiveJSID = *jsID
	}

	// Only emit delta for normal counts and resets (not jumps, which need operator confirmation)
	if anomaly != "jump" && delta > 0 {
		m.emitter.EmitCounterDelta(rp.ID, lineID, effectiveJSID, delta, newCount)
	}
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case int16:
		return int64(n), true
	case int:
		return int64(n), true
	case uint64:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint16:
		return int64(n), true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i, true
		}
		if f, err := n.Float64(); err == nil {
			return int64(f), true
		}
		return 0, false
	default:
		return 0, false
	}
}
