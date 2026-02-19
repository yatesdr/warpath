package store

// CounterSnapshot records a PLC counter reading.
type CounterSnapshot struct {
	ID                int64   `json:"id"`
	ReportingPointID  int64   `json:"reporting_point_id"`
	CountValue        int64   `json:"count_value"`
	Delta             int64   `json:"delta"`
	Anomaly           *string `json:"anomaly"`
	OperatorConfirmed bool    `json:"operator_confirmed"`
	RecordedAt        string  `json:"recorded_at"`
}

func (db *DB) InsertCounterSnapshot(rpID int64, countValue, delta int64, anomaly string, confirmed bool) (int64, error) {
	var anomalyPtr *string
	if anomaly != "" {
		anomalyPtr = &anomaly
	}
	res, err := db.Exec(`INSERT INTO counter_snapshots (reporting_point_id, count_value, delta, anomaly, operator_confirmed) VALUES (?, ?, ?, ?, ?)`,
		rpID, countValue, delta, anomalyPtr, confirmed)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ListUnconfirmedAnomalies() ([]CounterSnapshot, error) {
	rows, err := db.Query(`
		SELECT id, reporting_point_id, count_value, delta, anomaly, operator_confirmed, recorded_at
		FROM counter_snapshots
		WHERE anomaly = 'jump' AND operator_confirmed = 0
		ORDER BY recorded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []CounterSnapshot
	for rows.Next() {
		var s CounterSnapshot
		if err := rows.Scan(&s.ID, &s.ReportingPointID, &s.CountValue, &s.Delta, &s.Anomaly, &s.OperatorConfirmed, &s.RecordedAt); err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, rows.Err()
}

func (db *DB) ConfirmAnomaly(id int64) error {
	_, err := db.Exec(`UPDATE counter_snapshots SET operator_confirmed = 1 WHERE id = ?`, id)
	return err
}

func (db *DB) DismissAnomaly(id int64) error {
	_, err := db.Exec(`DELETE FROM counter_snapshots WHERE id = ? AND anomaly = 'jump' AND operator_confirmed = 0`, id)
	return err
}
