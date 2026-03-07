package store

// TagVerifyResult holds the result of a tag verification check.
type TagVerifyResult struct {
	Match    bool
	Expected string
	Detail   string
}

// VerifyTag performs best-effort QR tag verification for an order.
// It looks up the order's bin, learns new tags, and logs audit events.
// Returns match=true even on mismatch/missing (best-effort: never blocks orders).
func (db *DB) VerifyTag(orderUUID, tagID, location string) *TagVerifyResult {
	order, err := db.GetOrderByUUID(orderUUID)
	if err != nil {
		return &TagVerifyResult{Match: true, Detail: "order not found — accepting scan"}
	}

	if order.BinID == nil {
		return &TagVerifyResult{Match: true, Detail: "no bin assigned — accepting scan"}
	}

	bin, err := db.GetBin(*order.BinID)
	if err != nil {
		return &TagVerifyResult{Match: true, Detail: "bin not found — accepting scan"}
	}

	if bin.Label == "" {
		// Learn the tag on first scan by updating bin label
		bin.Label = tagID
		db.UpdateBin(bin)
		db.AppendAudit("bin", bin.ID, "tag_scanned", "", "tag learned from scan: "+tagID, "system")
		return &TagVerifyResult{Match: true, Detail: "tag learned: " + tagID}
	}

	if bin.Label == tagID {
		db.AppendAudit("bin", bin.ID, "tag_scanned", "", "tag verified at "+location, "system")
		return &TagVerifyResult{Match: true, Detail: "tag match"}
	}

	// Tag mismatch — best-effort: log but proceed
	db.AppendAudit("bin", bin.ID, "tag_mismatch", bin.Label, tagID, "system")
	return &TagVerifyResult{Match: false, Expected: bin.Label, Detail: "tag mismatch — proceeding (best-effort)"}
}
