package store

// ReportingPoint maps a PLC tag to a job style for counter tracking.
type ReportingPoint struct {
	ID         int64   `json:"id"`
	PLCName    string  `json:"plc_name"`
	TagName    string  `json:"tag_name"`
	JobStyleID int64   `json:"job_style_id"`
	LastCount  int64   `json:"last_count"`
	LastPollAt *string `json:"last_poll_at"`
	Enabled    bool    `json:"enabled"`
	LineID     *int64  `json:"line_id"`
}

func (db *DB) ListReportingPoints() ([]ReportingPoint, error) {
	rows, err := db.Query(`SELECT id, plc_name, tag_name, job_style_id, last_count, last_poll_at, enabled, line_id FROM reporting_points ORDER BY plc_name, tag_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rps []ReportingPoint
	for rows.Next() {
		var rp ReportingPoint
		if err := rows.Scan(&rp.ID, &rp.PLCName, &rp.TagName, &rp.JobStyleID, &rp.LastCount, &rp.LastPollAt, &rp.Enabled, &rp.LineID); err != nil {
			return nil, err
		}
		rps = append(rps, rp)
	}
	return rps, rows.Err()
}

func (db *DB) ListEnabledReportingPoints() ([]ReportingPoint, error) {
	rows, err := db.Query(`SELECT id, plc_name, tag_name, job_style_id, last_count, last_poll_at, enabled, line_id FROM reporting_points WHERE enabled = 1 ORDER BY plc_name, tag_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rps []ReportingPoint
	for rows.Next() {
		var rp ReportingPoint
		if err := rows.Scan(&rp.ID, &rp.PLCName, &rp.TagName, &rp.JobStyleID, &rp.LastCount, &rp.LastPollAt, &rp.Enabled, &rp.LineID); err != nil {
			return nil, err
		}
		rps = append(rps, rp)
	}
	return rps, rows.Err()
}

func (db *DB) GetReportingPoint(id int64) (*ReportingPoint, error) {
	rp := &ReportingPoint{}
	err := db.QueryRow(`SELECT id, plc_name, tag_name, job_style_id, last_count, last_poll_at, enabled, line_id FROM reporting_points WHERE id = ?`, id).
		Scan(&rp.ID, &rp.PLCName, &rp.TagName, &rp.JobStyleID, &rp.LastCount, &rp.LastPollAt, &rp.Enabled, &rp.LineID)
	if err != nil {
		return nil, err
	}
	return rp, nil
}

func (db *DB) CreateReportingPoint(plcName, tagName string, jobStyleID int64, lineID *int64) (int64, error) {
	res, err := db.Exec(`INSERT INTO reporting_points (plc_name, tag_name, job_style_id, line_id) VALUES (?, ?, ?, ?)`, plcName, tagName, jobStyleID, lineID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) UpdateReportingPoint(id int64, plcName, tagName string, jobStyleID int64, enabled bool, lineID *int64) error {
	_, err := db.Exec(`UPDATE reporting_points SET plc_name=?, tag_name=?, job_style_id=?, enabled=?, line_id=? WHERE id=?`, plcName, tagName, jobStyleID, enabled, lineID, id)
	return err
}

func (db *DB) UpdateReportingPointCounter(id int64, count int64) error {
	_, err := db.Exec(`UPDATE reporting_points SET last_count=?, last_poll_at=datetime('now','localtime') WHERE id=?`, count, id)
	return err
}

func (db *DB) DeleteReportingPoint(id int64) error {
	_, err := db.Exec(`DELETE FROM reporting_points WHERE id=?`, id)
	return err
}
