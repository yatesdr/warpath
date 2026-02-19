package store

import (
	"database/sql"
	"fmt"
	"time"
)

type Order struct {
	ID            int64      `json:"id"`
	EdgeUUID      string     `json:"edge_uuid"`
	StationID     string     `json:"station_id"`
	OrderType     string     `json:"order_type"`
	Status        string     `json:"status"`
	MaterialID    *int64     `json:"material_id,omitempty"`
	MaterialCode  string     `json:"material_code"`
	Quantity      float64    `json:"quantity"`
	PickupNode    string     `json:"pickup_node"`
	DeliveryNode  string     `json:"delivery_node"`
	VendorOrderID string     `json:"vendor_order_id"`
	VendorState   string     `json:"vendor_state"`
	RobotID       string     `json:"robot_id"`
	Priority      int        `json:"priority"`
	PayloadDesc   string     `json:"payload_desc"`
	ErrorDetail   string     `json:"error_detail"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	PayloadTypeID *int64     `json:"payload_type_id,omitempty"`
	PayloadID     *int64     `json:"payload_id,omitempty"`
}

type OrderHistory struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

const orderSelectCols = `id, edge_uuid, station_id, order_type, status, material_id, material_code, quantity, pickup_node, delivery_node, vendor_order_id, vendor_state, robot_id, priority, payload_desc, error_detail, created_at, updated_at, completed_at, payload_type_id, payload_id`

func scanOrder(row interface{ Scan(...any) error }) (*Order, error) {
	var o Order
	var materialID, payloadTypeID, payloadID sql.NullInt64
	var createdAt, updatedAt any
	var completedAt any

	err := row.Scan(&o.ID, &o.EdgeUUID, &o.StationID, &o.OrderType, &o.Status,
		&materialID, &o.MaterialCode, &o.Quantity,
		&o.PickupNode, &o.DeliveryNode, &o.VendorOrderID, &o.VendorState, &o.RobotID,
		&o.Priority, &o.PayloadDesc, &o.ErrorDetail, &createdAt, &updatedAt, &completedAt,
		&payloadTypeID, &payloadID)
	if err != nil {
		return nil, err
	}
	if materialID.Valid {
		o.MaterialID = &materialID.Int64
	}
	if payloadTypeID.Valid {
		o.PayloadTypeID = &payloadTypeID.Int64
	}
	if payloadID.Valid {
		o.PayloadID = &payloadID.Int64
	}
	o.CreatedAt = parseTime(createdAt)
	o.UpdatedAt = parseTime(updatedAt)
	o.CompletedAt = parseTimePtr(completedAt)
	return &o, nil
}

func scanOrders(rows *sql.Rows) ([]*Order, error) {
	var orders []*Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (db *DB) CreateOrder(o *Order) error {
	var matID, ptID, pID any
	if o.MaterialID != nil {
		matID = *o.MaterialID
	}
	if o.PayloadTypeID != nil {
		ptID = *o.PayloadTypeID
	}
	if o.PayloadID != nil {
		pID = *o.PayloadID
	}

	result, err := db.Exec(db.Q(`INSERT INTO orders (edge_uuid, station_id, order_type, status, material_id, material_code, quantity, pickup_node, delivery_node, priority, payload_desc, payload_type_id, payload_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		o.EdgeUUID, o.StationID, o.OrderType, o.Status,
		matID, o.MaterialCode, o.Quantity,
		o.PickupNode, o.DeliveryNode, o.Priority, o.PayloadDesc, ptID, pID)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("create order last id: %w", err)
	}
	o.ID = id
	return nil
}

func (db *DB) UpdateOrderStatus(id int64, status, detail string) error {
	_, err := db.Exec(db.Q(`UPDATE orders SET status=?, error_detail=?, updated_at=datetime('now','localtime') WHERE id=?`),
		status, detail, id)
	if err != nil {
		return err
	}
	_, err = db.Exec(db.Q(`INSERT INTO order_history (order_id, status, detail) VALUES (?, ?, ?)`),
		id, status, detail)
	return err
}

func (db *DB) UpdateOrderVendor(id int64, vendorOrderID, vendorState, robotID string) error {
	_, err := db.Exec(db.Q(`UPDATE orders SET vendor_order_id=?, vendor_state=?, robot_id=?, updated_at=datetime('now','localtime') WHERE id=?`),
		vendorOrderID, vendorState, robotID, id)
	return err
}

func (db *DB) UpdateOrderPickupNode(id int64, pickupNode string) error {
	_, err := db.Exec(db.Q(`UPDATE orders SET pickup_node=?, updated_at=datetime('now','localtime') WHERE id=?`),
		pickupNode, id)
	return err
}

func (db *DB) UpdateOrderDeliveryNode(id int64, deliveryNode string) error {
	_, err := db.Exec(db.Q(`UPDATE orders SET delivery_node=?, updated_at=datetime('now','localtime') WHERE id=?`),
		deliveryNode, id)
	return err
}

func (db *DB) CompleteOrder(id int64) error {
	_, err := db.Exec(db.Q(`UPDATE orders SET status='confirmed', completed_at=datetime('now','localtime'), updated_at=datetime('now','localtime') WHERE id=?`), id)
	if err != nil {
		return err
	}
	_, err = db.Exec(db.Q(`INSERT INTO order_history (order_id, status, detail) VALUES (?, 'confirmed', 'order confirmed')`), id)
	return err
}

func (db *DB) GetOrder(id int64) (*Order, error) {
	row := db.QueryRow(db.Q(fmt.Sprintf(`SELECT %s FROM orders WHERE id=?`, orderSelectCols)), id)
	return scanOrder(row)
}

func (db *DB) GetOrderByUUID(uuid string) (*Order, error) {
	row := db.QueryRow(db.Q(fmt.Sprintf(`SELECT %s FROM orders WHERE edge_uuid=? ORDER BY id DESC LIMIT 1`, orderSelectCols)), uuid)
	return scanOrder(row)
}

func (db *DB) GetOrderByVendorID(vendorOrderID string) (*Order, error) {
	row := db.QueryRow(db.Q(fmt.Sprintf(`SELECT %s FROM orders WHERE vendor_order_id=? LIMIT 1`, orderSelectCols)), vendorOrderID)
	return scanOrder(row)
}

func (db *DB) ListOrders(status string, limit int) ([]*Order, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = db.Query(db.Q(fmt.Sprintf(`SELECT %s FROM orders WHERE status=? ORDER BY id DESC LIMIT ?`, orderSelectCols)), status, limit)
	} else {
		rows, err = db.Query(db.Q(fmt.Sprintf(`SELECT %s FROM orders ORDER BY id DESC LIMIT ?`, orderSelectCols)), limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (db *DB) ListActiveOrders() ([]*Order, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`SELECT %s FROM orders WHERE status NOT IN ('confirmed', 'failed', 'cancelled') ORDER BY id DESC`, orderSelectCols)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (db *DB) ListOrderHistory(orderID int64) ([]*OrderHistory, error) {
	rows, err := db.Query(db.Q(`SELECT id, order_id, status, detail, created_at FROM order_history WHERE order_id=? ORDER BY id`), orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []*OrderHistory
	for rows.Next() {
		var h OrderHistory
		var createdAt any
		if err := rows.Scan(&h.ID, &h.OrderID, &h.Status, &h.Detail, &createdAt); err != nil {
			return nil, err
		}
		h.CreatedAt = parseTime(createdAt)
		history = append(history, &h)
	}
	return history, rows.Err()
}

func (db *DB) UpdateOrderPriority(id int64, priority int) error {
	_, err := db.Exec(db.Q(`UPDATE orders SET priority=?, updated_at=datetime('now','localtime') WHERE id=?`),
		priority, id)
	return err
}

func (db *DB) ListOrdersByStation(stationID string, limit int) ([]*Order, error) {
	rows, err := db.Query(db.Q(fmt.Sprintf(`SELECT %s FROM orders WHERE station_id=? ORDER BY id DESC LIMIT ?`, orderSelectCols)), stationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

// ListDispatchedVendorOrderIDs returns vendor order IDs for all non-terminal orders.
func (db *DB) ListDispatchedVendorOrderIDs() ([]string, error) {
	rows, err := db.Query(db.Q(`SELECT vendor_order_id FROM orders WHERE vendor_order_id != '' AND status IN ('dispatched', 'in_transit')`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
