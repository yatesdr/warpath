package store

import "database/sql"

// Order represents an E-Kanban order.
type Order struct {
	ID             int64    `json:"id"`
	UUID           string   `json:"uuid"`
	OrderType      string   `json:"order_type"`
	Status         string   `json:"status"`
	PayloadID      *int64   `json:"payload_id"`
	RetrieveEmpty  bool     `json:"retrieve_empty"`
	Quantity       float64  `json:"quantity"`
	DeliveryNode   string   `json:"delivery_node"`
	StagingNode    string   `json:"staging_node"`
	PickupNode     string   `json:"pickup_node"`
	LoadType       string   `json:"load_type"`
	TemplateID     *int64   `json:"template_id"`
	WaybillID      *string  `json:"waybill_id"`
	ExternalRef    *string  `json:"external_ref"`
	FinalCount     *float64 `json:"final_count"`
	CountConfirmed bool     `json:"count_confirmed"`
	ETA            *string  `json:"eta"`
	AutoConfirm    bool     `json:"auto_confirm"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`

	// Joined fields
	PayloadDesc     string `json:"payload_desc"`
	PayloadLocation string `json:"payload_location"`
	LineName        string `json:"line_name"`
}

// OrderHistory records a status transition.
type OrderHistory struct {
	ID        int64  `json:"id"`
	OrderID   int64  `json:"order_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"created_at"`
}

const orderSelectCols = `o.id, o.uuid, o.order_type, o.status, o.payload_id, o.retrieve_empty, o.quantity,
	o.delivery_node, o.staging_node, o.pickup_node, o.load_type,
	o.template_id, o.waybill_id, o.external_ref, o.final_count,
	o.count_confirmed, o.eta, o.auto_confirm, o.created_at, o.updated_at,
	COALESCE(p.description, ''), COALESCE(p.location, ''), COALESCE(pl.name, '')`

const orderJoin = `FROM orders o
	LEFT JOIN payloads p ON p.id = o.payload_id
	LEFT JOIN job_styles js ON js.id = p.job_style_id
	LEFT JOIN production_lines pl ON pl.id = js.line_id`

func (db *DB) ListOrders() ([]Order, error) {
	rows, err := db.Query(`SELECT ` + orderSelectCols + ` ` + orderJoin + ` ORDER BY o.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (db *DB) ListActiveOrders() ([]Order, error) {
	rows, err := db.Query(`SELECT ` + orderSelectCols + ` ` + orderJoin + `
		WHERE o.status NOT IN ('confirmed', 'cancelled')
		ORDER BY o.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (db *DB) ListActiveOrdersByLine(lineID int64) ([]Order, error) {
	rows, err := db.Query(`SELECT `+orderSelectCols+` `+orderJoin+`
		WHERE o.status NOT IN ('confirmed', 'cancelled')
		AND pl.id = ?
		ORDER BY o.created_at DESC`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

func scanOrders(rows *sql.Rows) ([]Order, error) {
	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.UUID, &o.OrderType, &o.Status, &o.PayloadID, &o.RetrieveEmpty, &o.Quantity,
			&o.DeliveryNode, &o.StagingNode, &o.PickupNode, &o.LoadType,
			&o.TemplateID, &o.WaybillID, &o.ExternalRef, &o.FinalCount,
			&o.CountConfirmed, &o.ETA, &o.AutoConfirm, &o.CreatedAt, &o.UpdatedAt,
			&o.PayloadDesc, &o.PayloadLocation, &o.LineName); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (db *DB) GetOrder(id int64) (*Order, error) {
	o := &Order{}
	err := db.QueryRow(`SELECT `+orderSelectCols+` `+orderJoin+` WHERE o.id = ?`, id).
		Scan(&o.ID, &o.UUID, &o.OrderType, &o.Status, &o.PayloadID, &o.RetrieveEmpty, &o.Quantity,
			&o.DeliveryNode, &o.StagingNode, &o.PickupNode, &o.LoadType,
			&o.TemplateID, &o.WaybillID, &o.ExternalRef, &o.FinalCount,
			&o.CountConfirmed, &o.ETA, &o.AutoConfirm, &o.CreatedAt, &o.UpdatedAt,
			&o.PayloadDesc, &o.PayloadLocation, &o.LineName)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (db *DB) GetOrderByUUID(uuid string) (*Order, error) {
	o := &Order{}
	err := db.QueryRow(`SELECT `+orderSelectCols+` `+orderJoin+` WHERE o.uuid = ?`, uuid).
		Scan(&o.ID, &o.UUID, &o.OrderType, &o.Status, &o.PayloadID, &o.RetrieveEmpty, &o.Quantity,
			&o.DeliveryNode, &o.StagingNode, &o.PickupNode, &o.LoadType,
			&o.TemplateID, &o.WaybillID, &o.ExternalRef, &o.FinalCount,
			&o.CountConfirmed, &o.ETA, &o.AutoConfirm, &o.CreatedAt, &o.UpdatedAt,
			&o.PayloadDesc, &o.PayloadLocation, &o.LineName)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (db *DB) CreateOrder(uuid, orderType string, payloadID *int64, retrieveEmpty bool, quantity float64, deliveryNode, stagingNode, pickupNode, loadType string, templateID *int64, autoConfirm bool) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO orders (uuid, order_type, payload_id, retrieve_empty, quantity, delivery_node, staging_node, pickup_node, load_type, template_id, auto_confirm)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid, orderType, payloadID, retrieveEmpty, quantity, deliveryNode, stagingNode, pickupNode, loadType, templateID, autoConfirm)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateOrderStatus(id int64, newStatus string) error {
	_, err := db.Exec(`UPDATE orders SET status=?, updated_at=datetime('now','localtime') WHERE id=?`, newStatus, id)
	return err
}

func (db *DB) UpdateOrderWaybill(id int64, waybillID, eta string) error {
	_, err := db.Exec(`UPDATE orders SET waybill_id=?, eta=?, updated_at=datetime('now','localtime') WHERE id=?`, waybillID, eta, id)
	return err
}

func (db *DB) UpdateOrderFinalCount(id int64, finalCount float64, confirmed bool) error {
	_, err := db.Exec(`UPDATE orders SET final_count=?, count_confirmed=?, updated_at=datetime('now','localtime') WHERE id=?`, finalCount, confirmed, id)
	return err
}

func (db *DB) UpdateOrderDeliveryNode(id int64, deliveryNode string) error {
	_, err := db.Exec(`UPDATE orders SET delivery_node=?, updated_at=datetime('now','localtime') WHERE id=?`, deliveryNode, id)
	return err
}

func (db *DB) ListKnownNodes() ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT location FROM payloads UNION SELECT DISTINCT staging_node FROM payloads WHERE staging_node != '' ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (db *DB) InsertOrderHistory(orderID int64, oldStatus, newStatus, detail string) error {
	_, err := db.Exec(`INSERT INTO order_history (order_id, old_status, new_status, detail) VALUES (?, ?, ?, ?)`,
		orderID, oldStatus, newStatus, detail)
	return err
}

func (db *DB) ListOrderHistory(orderID int64) ([]OrderHistory, error) {
	rows, err := db.Query(`SELECT id, order_id, old_status, new_status, detail, created_at FROM order_history WHERE order_id = ? ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []OrderHistory
	for rows.Next() {
		var h OrderHistory
		if err := rows.Scan(&h.ID, &h.OrderID, &h.OldStatus, &h.NewStatus, &h.Detail, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}
