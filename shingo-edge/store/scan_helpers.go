package store

import (
	"database/sql"
	"time"
)

const timeLayout = "2006-01-02 15:04:05"

func scanTime(s string) time.Time {
	t, _ := time.Parse(timeLayout, s)
	return t
}

func scanTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid {
		return nil
	}
	t, err := time.Parse(timeLayout, ns.String)
	if err != nil {
		return nil
	}
	return &t
}
