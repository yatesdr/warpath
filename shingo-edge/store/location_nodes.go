package store

// LocationNode represents a physical factory location.
type LocationNode struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Process     string `json:"process"`
	Description string `json:"description"`
}

func (db *DB) ListLocationNodes() ([]LocationNode, error) {
	rows, err := db.Query("SELECT id, node_id, process, description FROM location_nodes ORDER BY node_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []LocationNode
	for rows.Next() {
		var n LocationNode
		if err := rows.Scan(&n.ID, &n.NodeID, &n.Process, &n.Description); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (db *DB) CreateLocationNode(nodeID, process, description string) (int64, error) {
	res, err := db.Exec("INSERT INTO location_nodes (node_id, process, description) VALUES (?, ?, ?)", nodeID, process, description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateLocationNode(id int64, nodeID, process, description string) error {
	_, err := db.Exec("UPDATE location_nodes SET node_id = ?, process = ?, description = ? WHERE id = ?", nodeID, process, description, id)
	return err
}

func (db *DB) DeleteLocationNode(id int64) error {
	_, err := db.Exec("DELETE FROM location_nodes WHERE id = ?", id)
	return err
}
