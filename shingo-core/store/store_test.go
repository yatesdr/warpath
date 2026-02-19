package store

import (
	"os"
	"path/filepath"
	"testing"

	"shingocore/config"
)

// testDB creates a temporary SQLite database for testing.
func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(&config.DatabaseConfig{
		Driver: "sqlite",
		SQLite: config.SQLiteConfig{Path: dbPath},
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})
	return db
}

// --- Node tests ---

func TestNodeCRUD(t *testing.T) {
	db := testDB(t)

	n := &Node{Name: "STORAGE-A1", VendorLocation: "Loc-01", NodeType: "storage", Zone: "A", Capacity: 10, Enabled: true}
	if err := db.CreateNode(n); err != nil {
		t.Fatalf("create: %v", err)
	}
	if n.ID == 0 {
		t.Fatal("ID should be assigned")
	}

	got, err := db.GetNode(n.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "STORAGE-A1" {
		t.Errorf("Name = %q, want %q", got.Name, "STORAGE-A1")
	}
	if got.VendorLocation != "Loc-01" {
		t.Errorf("VendorLocation = %q, want %q", got.VendorLocation, "Loc-01")
	}
	if got.Capacity != 10 {
		t.Errorf("Capacity = %d, want 10", got.Capacity)
	}
	if !got.Enabled {
		t.Error("Enabled should be true")
	}

	// Update
	got.Capacity = 20
	got.Zone = "B"
	if err := db.UpdateNode(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	got2, _ := db.GetNode(n.ID)
	if got2.Capacity != 20 {
		t.Errorf("Capacity after update = %d, want 20", got2.Capacity)
	}
	if got2.Zone != "B" {
		t.Errorf("Zone after update = %q, want %q", got2.Zone, "B")
	}

	// GetByName
	got3, err := db.GetNodeByName("STORAGE-A1")
	if err != nil {
		t.Fatalf("getByName: %v", err)
	}
	if got3.ID != n.ID {
		t.Errorf("getByName ID = %d, want %d", got3.ID, n.ID)
	}

	// List
	db.CreateNode(&Node{Name: "LINE1-IN", VendorLocation: "Loc-02", NodeType: "line_side", Enabled: true})
	nodes, err := db.ListNodes()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("len = %d, want 2", len(nodes))
	}

	// ListByType
	storageNodes, _ := db.ListNodesByType("storage")
	if len(storageNodes) != 1 {
		t.Errorf("storage count = %d, want 1", len(storageNodes))
	}

	// Delete
	if err := db.DeleteNode(n.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = db.GetNode(n.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// --- Material tests ---

func TestMaterialCRUD(t *testing.T) {
	db := testDB(t)

	m := &Material{Code: "PART-A", Description: "Steel bracket", Unit: "ea"}
	if err := db.CreateMaterial(m); err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("ID should be assigned")
	}

	got, err := db.GetMaterial(m.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Code != "PART-A" {
		t.Errorf("Code = %q, want %q", got.Code, "PART-A")
	}
	if got.Description != "Steel bracket" {
		t.Errorf("Description = %q, want %q", got.Description, "Steel bracket")
	}

	// GetByCode
	got2, err := db.GetMaterialByCode("PART-A")
	if err != nil {
		t.Fatalf("getByCode: %v", err)
	}
	if got2.ID != m.ID {
		t.Errorf("getByCode ID = %d, want %d", got2.ID, m.ID)
	}

	// Update
	got.Description = "Updated"
	if err := db.UpdateMaterial(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	got3, _ := db.GetMaterial(m.ID)
	if got3.Description != "Updated" {
		t.Errorf("Description after update = %q, want %q", got3.Description, "Updated")
	}

	// List
	db.CreateMaterial(&Material{Code: "PART-B", Unit: "kg"})
	materials, _ := db.ListMaterials()
	if len(materials) != 2 {
		t.Errorf("len = %d, want 2", len(materials))
	}

	// Delete
	db.DeleteMaterial(m.ID)
	_, err = db.GetMaterial(m.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// --- Order tests ---

func TestOrderCRUD(t *testing.T) {
	db := testDB(t)

	matID := int64(0)
	o := &Order{
		EdgeUUID:  "uuid-1",
		StationID:    "line-1",
		OrderType:    "retrieve",
		Status:       "pending",
		MaterialCode: "PART-A",
		Quantity:     1.0,
		DeliveryNode: "LINE1-IN",
	}
	if err := db.CreateOrder(o); err != nil {
		t.Fatalf("create: %v", err)
	}
	if o.ID == 0 {
		t.Fatal("ID should be assigned")
	}
	_ = matID // not used in this test

	// Get
	got, err := db.GetOrder(o.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.EdgeUUID != "uuid-1" {
		t.Errorf("EdgeUUID = %q, want %q", got.EdgeUUID, "uuid-1")
	}
	if got.Status != "pending" {
		t.Errorf("Status = %q, want %q", got.Status, "pending")
	}

	// GetByUUID
	got2, err := db.GetOrderByUUID("uuid-1")
	if err != nil {
		t.Fatalf("getByUUID: %v", err)
	}
	if got2.ID != o.ID {
		t.Errorf("getByUUID ID = %d, want %d", got2.ID, o.ID)
	}

	// UpdateStatus (also creates history)
	db.UpdateOrderStatus(o.ID, "dispatched", "sent to RDS")
	got3, _ := db.GetOrder(o.ID)
	if got3.Status != "dispatched" {
		t.Errorf("Status after update = %q, want %q", got3.Status, "dispatched")
	}

	// Check history
	history, _ := db.ListOrderHistory(o.ID)
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].Status != "dispatched" {
		t.Errorf("history status = %q, want %q", history[0].Status, "dispatched")
	}

	// UpdateVendor
	db.UpdateOrderVendor(o.ID, "rds-123", "RUNNING", "AMB-01")
	got4, _ := db.GetOrder(o.ID)
	if got4.VendorOrderID != "rds-123" {
		t.Errorf("VendorOrderID = %q, want %q", got4.VendorOrderID, "rds-123")
	}
	if got4.RobotID != "AMB-01" {
		t.Errorf("RobotID = %q, want %q", got4.RobotID, "AMB-01")
	}

	// GetByVendorID
	got5, err := db.GetOrderByVendorID("rds-123")
	if err != nil {
		t.Fatalf("getByVendorID: %v", err)
	}
	if got5.ID != o.ID {
		t.Errorf("getByVendorID ID = %d, want %d", got5.ID, o.ID)
	}

	// Complete
	db.CompleteOrder(o.ID)
	got6, _ := db.GetOrder(o.ID)
	if got6.Status != "confirmed" {
		t.Errorf("Status after complete = %q, want %q", got6.Status, "confirmed")
	}
	if got6.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestListOrders(t *testing.T) {
	db := testDB(t)

	db.CreateOrder(&Order{EdgeUUID: "u1", Status: "pending"})
	db.CreateOrder(&Order{EdgeUUID: "u2", Status: "confirmed"})
	db.CreateOrder(&Order{EdgeUUID: "u3", Status: "pending"})

	// All
	all, _ := db.ListOrders("", 10)
	if len(all) != 3 {
		t.Errorf("all len = %d, want 3", len(all))
	}

	// Filtered
	pending, _ := db.ListOrders("pending", 10)
	if len(pending) != 2 {
		t.Errorf("pending len = %d, want 2", len(pending))
	}

	// Active
	active, _ := db.ListActiveOrders()
	if len(active) != 2 {
		t.Errorf("active len = %d, want 2", len(active))
	}
}

func TestListDispatchedVendorOrderIDs(t *testing.T) {
	db := testDB(t)

	o1 := &Order{EdgeUUID: "u1", Status: "dispatched"}
	o2 := &Order{EdgeUUID: "u2", Status: "in_transit"}
	o3 := &Order{EdgeUUID: "u3", Status: "confirmed"}
	db.CreateOrder(o1)
	db.CreateOrder(o2)
	db.CreateOrder(o3)
	db.UpdateOrderVendor(o1.ID, "rds-1", "CREATED", "")
	db.UpdateOrderVendor(o2.ID, "rds-2", "RUNNING", "")
	db.UpdateOrderVendor(o3.ID, "rds-3", "FINISHED", "")

	ids, err := db.ListDispatchedVendorOrderIDs()
	if err != nil {
		t.Fatalf("list dispatched: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("len = %d, want 2", len(ids))
	}
}

// --- Outbox tests ---

func TestOutboxCRUD(t *testing.T) {
	db := testDB(t)

	// Enqueue
	if err := db.EnqueueOutbox("shingo.dispatch", []byte(`{"test":true}`), "order.ack", "line-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	db.EnqueueOutbox("shingo.dispatch", []byte(`{"test":2}`), "order.update", "line-2")

	// List pending
	msgs, err := db.ListPendingOutbox(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len = %d, want 2", len(msgs))
	}
	if msgs[0].Topic != "shingo.dispatch" {
		t.Errorf("topic = %q, want %q", msgs[0].Topic, "shingo.dispatch")
	}
	if msgs[0].MsgType != "order.ack" {
		t.Errorf("msg_type = %q, want %q", msgs[0].MsgType, "order.ack")
	}

	// Ack
	db.AckOutbox(msgs[0].ID)
	msgs2, _ := db.ListPendingOutbox(10)
	if len(msgs2) != 1 {
		t.Errorf("pending after ack = %d, want 1", len(msgs2))
	}

	// Increment retries
	db.IncrementOutboxRetries(msgs2[0].ID)
	msgs3, _ := db.ListPendingOutbox(10)
	if msgs3[0].Retries != 1 {
		t.Errorf("retries = %d, want 1", msgs3[0].Retries)
	}
}

// --- Audit tests ---

func TestAuditLog(t *testing.T) {
	db := testDB(t)

	db.AppendAudit("order", 1, "created", "", "new order", "system")
	db.AppendAudit("order", 1, "dispatched", "pending", "dispatched", "system")
	db.AppendAudit("node", 2, "updated", "", "S1", "admin")

	// List all
	entries, err := db.ListAuditLog(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("len = %d, want 3", len(entries))
	}
	// Most recent first
	if entries[0].Action != "updated" {
		t.Errorf("first entry action = %q, want %q", entries[0].Action, "updated")
	}

	// List by entity
	orderEntries, _ := db.ListEntityAudit("order", 1)
	if len(orderEntries) != 2 {
		t.Errorf("order entries = %d, want 2", len(orderEntries))
	}
}

// --- Correction tests ---

func TestCorrectionCRUD(t *testing.T) {
	db := testDB(t)

	node := &Node{Name: "S1", VendorLocation: "Loc-01", NodeType: "storage", Enabled: true}
	db.CreateNode(node)

	c := &Correction{
		CorrectionType: "add",
		NodeID:         node.ID,
		Quantity:       5.0,
		Reason:         "physical count mismatch",
		Actor:          "admin",
	}
	if err := db.CreateCorrection(c); err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("ID should be assigned")
	}

	corrections, err := db.ListCorrections(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(corrections) != 1 {
		t.Fatalf("len = %d, want 1", len(corrections))
	}
	if corrections[0].CorrectionType != "add" {
		t.Errorf("type = %q, want %q", corrections[0].CorrectionType, "add")
	}
	if corrections[0].Reason != "physical count mismatch" {
		t.Errorf("reason = %q, want %q", corrections[0].Reason, "physical count mismatch")
	}
}

// --- Dialect tests ---

func TestRebind(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SELECT * FROM t WHERE a=? AND b=?", "SELECT * FROM t WHERE a=$1 AND b=$2"},
		{"INSERT INTO t (a) VALUES (?)", "INSERT INTO t (a) VALUES ($1)"},
		{"SELECT 1", "SELECT 1"},
	}
	for _, tt := range tests {
		got := Rebind(tt.input)
		if got != tt.want {
			t.Errorf("Rebind(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
