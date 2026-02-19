package engine

import (
	"log"
	"time"

	"shingocore/config"
	"shingocore/dispatch"
	"shingocore/fleet"
	"shingocore/messaging"
	"shingocore/nodestate"
	"shingocore/store"
)

type LogFunc func(format string, args ...any)

type Config struct {
	AppConfig  *config.Config
	ConfigPath string
	DB         *store.DB
	Fleet      fleet.Backend
	NodeState  *nodestate.Manager
	MsgClient  *messaging.Client
	LogFunc    LogFunc
	Debug      bool
}

type Engine struct {
	cfg            *config.Config
	configPath     string
	db             *store.DB
	fleet          fleet.Backend
	nodeState      *nodestate.Manager
	msgClient      *messaging.Client
	dispatcher     *dispatch.Dispatcher
	tracker        fleet.OrderTracker
	Events         *EventBus
	logFn          LogFunc
	stopChan       chan struct{}
	fleetConnected bool
	msgConnected   bool
}

func New(c Config) *Engine {
	logFn := c.LogFunc
	if logFn == nil {
		logFn = log.Printf
	}
	return &Engine{
		cfg:        c.AppConfig,
		configPath: c.ConfigPath,
		db:         c.DB,
		fleet:      c.Fleet,
		nodeState:  c.NodeState,
		msgClient:  c.MsgClient,
		Events:     NewEventBus(),
		logFn:      logFn,
		stopChan:   make(chan struct{}),
	}
}

func (e *Engine) Start() {
	// Create emitter adapters
	de := &dispatchEmitter{bus: e.Events}
	pe := &pollerEmitter{bus: e.Events}

	// Create dispatcher
	e.dispatcher = dispatch.NewDispatcher(
		e.db,
		e.fleet,
		de,
		e.cfg.Messaging.StationID,
		e.cfg.Messaging.DispatchTopic,
	)

	// Initialize tracker if backend supports it
	if tb, ok := e.fleet.(fleet.TrackingBackend); ok {
		tb.InitTracker(pe, &orderResolver{db: e.db})
		e.tracker = tb.Tracker()
	}

	// Wire event handlers
	e.wireEventHandlers()

	// Load active vendor orders into tracker
	e.loadActiveOrders()

	// Start tracker
	if e.tracker != nil {
		e.tracker.Start()
	}

	// Emit initial connection status
	e.checkConnectionStatus()

	// Start periodic connection health check
	go e.connectionHealthLoop()

	e.logFn("engine: started")
}

func (e *Engine) Stop() {
	select {
	case e.stopChan <- struct{}{}:
	default:
	}
	if e.tracker != nil {
		e.tracker.Stop()
	}
	e.logFn("engine: stopped")
}

// Accessors
func (e *Engine) DB() *store.DB                    { return e.db }
func (e *Engine) AppConfig() *config.Config        { return e.cfg }
func (e *Engine) ConfigPath() string               { return e.configPath }
func (e *Engine) Dispatcher() *dispatch.Dispatcher  { return e.dispatcher }
func (e *Engine) NodeState() *nodestate.Manager     { return e.nodeState }
func (e *Engine) Tracker() fleet.OrderTracker       { return e.tracker }
func (e *Engine) Fleet() fleet.Backend              { return e.fleet }
func (e *Engine) MsgClient() *messaging.Client      { return e.msgClient }

func (e *Engine) checkConnectionStatus() {
	// Fleet
	if err := e.fleet.Ping(); err == nil {
		if !e.fleetConnected {
			e.fleetConnected = true
			e.Events.Emit(Event{Type: EventFleetConnected, Payload: ConnectionEvent{Detail: e.fleet.Name() + " connected"}})
		}
	} else {
		if e.fleetConnected {
			e.fleetConnected = false
			e.Events.Emit(Event{Type: EventFleetDisconnected, Payload: ConnectionEvent{Detail: err.Error()}})
		}
	}

	// Messaging
	if e.msgClient.IsConnected() {
		if !e.msgConnected {
			e.msgConnected = true
			e.Events.Emit(Event{Type: EventMessagingConnected, Payload: ConnectionEvent{Detail: "messaging connected"}})
		}
	} else {
		if e.msgConnected {
			e.msgConnected = false
			e.Events.Emit(Event{Type: EventMessagingDisconnected, Payload: ConnectionEvent{Detail: "messaging disconnected"}})
		}
	}
}

func (e *Engine) connectionHealthLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.checkConnectionStatus()
		}
	}
}

func (e *Engine) loadActiveOrders() {
	if e.tracker == nil {
		return
	}
	ids, err := e.db.ListDispatchedVendorOrderIDs()
	if err != nil {
		e.logFn("engine: load active orders: %v", err)
		return
	}
	for _, id := range ids {
		e.tracker.Track(id)
	}
	if len(ids) > 0 {
		e.logFn("engine: loaded %d active vendor orders into tracker", len(ids))
	}
}

// ReconfigureFleet applies fleet config changes live.
func (e *Engine) ReconfigureFleet() {
	e.fleet.Reconfigure(fleet.ReconfigureParams{
		BaseURL: e.cfg.RDS.BaseURL,
		Timeout: e.cfg.RDS.Timeout,
	})
	e.logFn("engine: fleet reconfigured (%s)", e.fleet.Name())
	e.checkConnectionStatus()
}

// ReconfigureMessaging reconnects messaging with current config.
func (e *Engine) ReconfigureMessaging() {
	if err := e.msgClient.Reconfigure(&e.cfg.Messaging); err != nil {
		e.logFn("engine: messaging reconfigure error: %v", err)
	} else {
		e.logFn("engine: messaging reconfigured")
	}
	e.checkConnectionStatus()
}

// SyncScenePoints persists fleet scene areas to the database.
// Returns the total number of points synced and a map of bin location instanceName → areaName.
func (e *Engine) SyncScenePoints(areas []fleet.SceneArea) (int, map[string]string) {
	locationSet := make(map[string]string)
	total := 0
	for _, area := range areas {
		e.db.DeleteScenePointsByArea(area.Name)
		for _, ap := range area.AdvancedPoints {
			sp := &store.ScenePoint{
				AreaName:       area.Name,
				InstanceName:   ap.InstanceName,
				ClassName:      ap.ClassName,
				Label:          ap.Label,
				PosX:           ap.PosX,
				PosY:           ap.PosY,
				PosZ:           ap.PosZ,
				Dir:            ap.Dir,
				PropertiesJSON: ap.PropertiesJSON,
			}
			e.db.UpsertScenePoint(sp)
			total++
		}
		for _, bin := range area.BinLocations {
			locationSet[bin.InstanceName] = area.Name
			sp := &store.ScenePoint{
				AreaName:       area.Name,
				InstanceName:   bin.InstanceName,
				ClassName:      bin.ClassName,
				PointName:      bin.PointName,
				GroupName:      bin.GroupName,
				PosX:           bin.PosX,
				PosY:           bin.PosY,
				PosZ:           bin.PosZ,
				PropertiesJSON: bin.PropertiesJSON,
			}
			e.db.UpsertScenePoint(sp)
			total++
		}
	}
	return total, locationSet
}

// SyncFleetNodes creates nodes for new scene locations and removes nodes no longer in the scene.
// Returns the number of nodes created and deleted.
func (e *Engine) SyncFleetNodes(locationSet map[string]string) (created, deleted int) {
	// Create nodes for locations not yet in DB
	for instanceName, areaName := range locationSet {
		if _, err := e.db.GetNodeByVendorLocation(instanceName); err == nil {
			continue
		}
		node := &store.Node{
			Name:           instanceName,
			VendorLocation: instanceName,
			NodeType:       "storage",
			Zone:           areaName,
			Capacity:       1,
			Enabled:        true,
		}
		if err := e.db.CreateNode(node); err != nil {
			continue
		}
		e.nodeState.RefreshNodeMeta(node.ID)
		e.Events.Emit(Event{Type: EventNodeUpdated, Payload: NodeUpdatedEvent{
			NodeID: node.ID, NodeName: node.Name, Action: "created",
		}})
		created++
	}

	// Delete nodes not present in current scene
	nodes, _ := e.db.ListNodes()
	for _, n := range nodes {
		if _, inScene := locationSet[n.VendorLocation]; !inScene {
			e.db.DeleteNode(n.ID)
			e.Events.Emit(Event{Type: EventNodeUpdated, Payload: NodeUpdatedEvent{
				NodeID: n.ID, NodeName: n.Name, Action: "deleted",
			}})
			deleted++
		}
	}

	// Update zones on remaining nodes
	e.UpdateNodeZones(locationSet, true)
	return
}

// UpdateNodeZones updates node zones from a location→area map.
// If overwrite is true, updates zone whenever it differs; if false, only fills empty zones.
func (e *Engine) UpdateNodeZones(locationSet map[string]string, overwrite bool) {
	nodes, _ := e.db.ListNodes()
	for _, n := range nodes {
		if n.VendorLocation == "" {
			continue
		}
		zone, ok := locationSet[n.VendorLocation]
		if !ok {
			continue
		}
		if !overwrite && n.Zone != "" {
			continue
		}
		if n.Zone == zone {
			continue
		}
		n.Zone = zone
		e.db.UpdateNode(n)
		e.Events.Emit(Event{Type: EventNodeUpdated, Payload: NodeUpdatedEvent{
			NodeID: n.ID, NodeName: n.Name, Action: "updated",
		}})
	}
}

// orderResolver implements fleet.OrderIDResolver.
type orderResolver struct {
	db *store.DB
}

func (r *orderResolver) ResolveVendorOrderID(vendorOrderID string) (int64, error) {
	order, err := r.db.GetOrderByVendorID(vendorOrderID)
	if err != nil {
		return 0, err
	}
	return order.ID, nil
}
