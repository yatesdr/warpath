package store

// ProductionLine represents a physical production line.
type ProductionLine struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	ActiveJobStyleID *int64 `json:"active_job_style_id"`
	CreatedAt        string `json:"created_at"`
}

func (db *DB) ListProductionLines() ([]ProductionLine, error) {
	rows, err := db.Query(`SELECT id, name, description, active_job_style_id, created_at FROM production_lines ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []ProductionLine
	for rows.Next() {
		var l ProductionLine
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.ActiveJobStyleID, &l.CreatedAt); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func (db *DB) GetProductionLine(id int64) (*ProductionLine, error) {
	l := &ProductionLine{}
	err := db.QueryRow(`SELECT id, name, description, active_job_style_id, created_at FROM production_lines WHERE id = ?`, id).
		Scan(&l.ID, &l.Name, &l.Description, &l.ActiveJobStyleID, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (db *DB) CreateProductionLine(name, description string) (int64, error) {
	res, err := db.Exec(`INSERT INTO production_lines (name, description) VALUES (?, ?)`, name, description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateProductionLine(id int64, name, description string) error {
	_, err := db.Exec(`UPDATE production_lines SET name=?, description=? WHERE id=?`, name, description, id)
	return err
}

func (db *DB) DeleteProductionLine(id int64) error {
	_, err := db.Exec(`DELETE FROM production_lines WHERE id=?`, id)
	return err
}

func (db *DB) SetActiveJobStyle(lineID int64, jobStyleID *int64) error {
	_, err := db.Exec(`UPDATE production_lines SET active_job_style_id=? WHERE id=?`, jobStyleID, lineID)
	return err
}

func (db *DB) GetActiveJobStyleID(lineID int64) (*int64, error) {
	var id *int64
	err := db.QueryRow(`SELECT active_job_style_id FROM production_lines WHERE id = ?`, lineID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return id, nil
}
