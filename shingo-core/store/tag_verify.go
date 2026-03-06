package store

// TagVerifyResult holds the result of a tag verification check.
type TagVerifyResult struct {
	Match    bool
	Expected string
	Detail   string
}

// VerifyTag performs best-effort QR tag verification for an order.
// It looks up the claimed payload's bin, learns new tags, and logs events.
// Returns match=true even on mismatch/missing (best-effort: never blocks orders).
func (db *DB) VerifyTag(orderUUID, tagID, location string) *TagVerifyResult {
	order, err := db.GetOrderByUUID(orderUUID)
	if err != nil {
		return &TagVerifyResult{Match: true, Detail: "order not found — accepting scan"}
	}

	if order.PayloadID == nil {
		return &TagVerifyResult{Match: true, Detail: "no payload tracking — accepting scan"}
	}

	payload, err := db.GetPayload(*order.PayloadID)
	if err != nil {
		return &TagVerifyResult{Match: true, Detail: "payload not found — accepting scan"}
	}

	if payload.BinID == nil {
		return &TagVerifyResult{Match: true, Detail: "no bin assigned — accepting scan"}
	}

	bin, err := db.GetBin(*payload.BinID)
	if err != nil {
		return &TagVerifyResult{Match: true, Detail: "bin not found — accepting scan"}
	}

	if bin.Label == "" {
		// Learn the tag on first scan by updating bin label
		bin.Label = tagID
		db.UpdateBin(bin)
		db.CreatePayloadEvent(&PayloadEvent{
			PayloadID: payload.ID, EventType: PayloadEventTagScanned,
			Detail: "tag learned from scan: " + tagID, Actor: "system",
		})
		return &TagVerifyResult{Match: true, Detail: "tag learned: " + tagID}
	}

	if bin.Label == tagID {
		db.CreatePayloadEvent(&PayloadEvent{
			PayloadID: payload.ID, EventType: PayloadEventTagScanned,
			Detail: "tag verified at " + location, Actor: "system",
		})
		return &TagVerifyResult{Match: true, Detail: "tag match"}
	}

	// Tag mismatch — best-effort: log but proceed
	db.CreatePayloadEvent(&PayloadEvent{
		PayloadID: payload.ID, EventType: PayloadEventTagMismatch,
		Detail: "expected " + bin.Label + " got " + tagID, Actor: "system",
	})
	return &TagVerifyResult{Match: false, Expected: bin.Label, Detail: "tag mismatch — proceeding (best-effort)"}
}
