package store

// LocationNode represents a physical factory location tied to a production line.
type LocationNode struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	LineID      int64  `json:"line_id"`
	Description string `json:"description"`
}

func (db *DB) ListLocationNodes() ([]LocationNode, error) {
	rows, err := db.Query("SELECT id, node_id, COALESCE(line_id, 0), description FROM location_nodes ORDER BY node_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []LocationNode
	for rows.Next() {
		var n LocationNode
		if err := rows.Scan(&n.ID, &n.NodeID, &n.LineID, &n.Description); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (db *DB) ListLocationNodesByLine(lineID int64) ([]LocationNode, error) {
	rows, err := db.Query("SELECT id, node_id, COALESCE(line_id, 0), description FROM location_nodes WHERE line_id = ? ORDER BY node_id", lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []LocationNode
	for rows.Next() {
		var n LocationNode
		if err := rows.Scan(&n.ID, &n.NodeID, &n.LineID, &n.Description); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ListKnownNodes returns distinct location and staging_node values from payloads.
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

func (db *DB) CreateLocationNode(nodeID string, lineID int64, description string) (int64, error) {
	res, err := db.Exec("INSERT INTO location_nodes (node_id, line_id, description) VALUES (?, ?, ?)", nodeID, lineID, description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateLocationNode(id int64, nodeID string, lineID int64, description string) error {
	_, err := db.Exec("UPDATE location_nodes SET node_id = ?, line_id = ?, description = ? WHERE id = ?", nodeID, lineID, description, id)
	return err
}

func (db *DB) DeleteLocationNode(id int64) error {
	_, err := db.Exec("DELETE FROM location_nodes WHERE id = ?", id)
	return err
}
