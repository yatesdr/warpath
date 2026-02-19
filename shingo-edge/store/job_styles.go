package store

// JobStyle represents a product/recipe style that maps to a BOM.
type JobStyle struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LineID      *int64 `json:"line_id"`
	CreatedAt   string `json:"created_at"`
}

func (db *DB) ListJobStyles() ([]JobStyle, error) {
	rows, err := db.Query(`SELECT id, name, description, line_id, created_at FROM job_styles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var styles []JobStyle
	for rows.Next() {
		var s JobStyle
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.LineID, &s.CreatedAt); err != nil {
			return nil, err
		}
		styles = append(styles, s)
	}
	return styles, rows.Err()
}

func (db *DB) ListJobStylesByLine(lineID int64) ([]JobStyle, error) {
	rows, err := db.Query(`SELECT id, name, description, line_id, created_at FROM job_styles WHERE line_id = ? ORDER BY name`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var styles []JobStyle
	for rows.Next() {
		var s JobStyle
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.LineID, &s.CreatedAt); err != nil {
			return nil, err
		}
		styles = append(styles, s)
	}
	return styles, rows.Err()
}

func (db *DB) GetJobStyleByName(name string) (*JobStyle, error) {
	s := &JobStyle{}
	err := db.QueryRow(`SELECT id, name, description, line_id, created_at FROM job_styles WHERE name = ?`, name).
		Scan(&s.ID, &s.Name, &s.Description, &s.LineID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *DB) GetJobStyle(id int64) (*JobStyle, error) {
	s := &JobStyle{}
	err := db.QueryRow(`SELECT id, name, description, line_id, created_at FROM job_styles WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Description, &s.LineID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *DB) CreateJobStyle(name, description string, lineID int64) (int64, error) {
	res, err := db.Exec(`INSERT INTO job_styles (name, description, line_id) VALUES (?, ?, ?)`, name, description, lineID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateJobStyle(id int64, name, description string, lineID int64) error {
	_, err := db.Exec(`UPDATE job_styles SET name=?, description=?, line_id=? WHERE id=?`, name, description, lineID, id)
	return err
}

func (db *DB) DeleteJobStyle(id int64) error {
	_, err := db.Exec(`DELETE FROM job_styles WHERE id=?`, id)
	return err
}
