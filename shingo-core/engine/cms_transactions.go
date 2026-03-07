package engine

import (
	"shingocore/store"
)

// FindCMSBoundary walks up the parent chain from nodeID to find the nearest
// synthetic ancestor (or self) that has CMS transaction logging enabled.
// Default: parentless synthetic nodes are enabled unless property is "false".
// Child synthetic nodes are disabled unless property is explicitly "true".
func (e *Engine) FindCMSBoundary(nodeID int64) *store.Node {
	visited := make(map[int64]bool)
	currentID := nodeID
	for {
		if visited[currentID] {
			return nil
		}
		visited[currentID] = true

		node, err := e.db.GetNode(currentID)
		if err != nil {
			return nil
		}

		if node.IsSynthetic {
			prop := e.db.GetNodeProperty(node.ID, "log_cms_transactions")
			if node.ParentID == nil {
				if prop != "false" {
					return node
				}
			} else {
				if prop == "true" {
					return node
				}
			}
		}

		if node.ParentID == nil {
			return nil
		}
		currentID = *node.ParentID
	}
}

// txnType returns "increase" or "decrease" based on the sign of delta.
func txnType(delta int64) string {
	if delta >= 0 {
		return "increase"
	}
	return "decrease"
}

// RecordMovementTransactions logs CMS transactions when a bin moves between
// different CMS boundaries. Delta is signed: negative = leaving, positive = arriving.
// QtyBefore/QtyAfter reflect the boundary-level total for each CATID.
func (e *Engine) RecordMovementTransactions(ev BinUpdatedEvent) {
	var srcBoundary, dstBoundary *store.Node
	if ev.FromNodeID != 0 {
		srcBoundary = e.FindCMSBoundary(ev.FromNodeID)
	}
	if ev.ToNodeID != 0 {
		dstBoundary = e.FindCMSBoundary(ev.ToNodeID)
	}

	srcID := int64(0)
	dstID := int64(0)
	if srcBoundary != nil {
		srcID = srcBoundary.ID
	}
	if dstBoundary != nil {
		dstID = dstBoundary.ID
	}
	if srcID == dstID {
		return
	}

	// Get the bin and parse its manifest
	bin, err := e.db.GetBin(ev.BinID)
	if err != nil {
		return
	}

	parsed, _ := bin.ParseManifest()
	if parsed == nil || len(parsed.Items) == 0 {
		return
	}

	var orderID *int64
	if bin.ClaimedBy != nil && *bin.ClaimedBy != 0 {
		orderID = bin.ClaimedBy
	}

	var txns []*store.CMSTransaction

	// Source boundary: bin leaving → negative delta.
	if srcBoundary != nil {
		totals := e.db.SumCatIDsAtBoundary(srcBoundary.ID)
		for _, m := range parsed.Items {
			if m.Quantity <= 0 {
				continue
			}
			delta := -m.Quantity
			qtyAfter := totals[m.CatID]
			qtyBefore := qtyAfter + m.Quantity
			txns = append(txns, &store.CMSTransaction{
				NodeID:      srcBoundary.ID,
				NodeName:    srcBoundary.Name,
				TxnType:     txnType(delta),
				CatID:       m.CatID,
				Delta:       delta,
				QtyBefore:   qtyBefore,
				QtyAfter:    qtyAfter,
				BinID:       &bin.ID,
				BinLabel:    bin.Label,
				PayloadCode: bin.PayloadCode,
				SourceType:  "movement",
				OrderID:     orderID,
				Notes:       "auto-log",
			})
		}
	}

	// Dest boundary: bin arriving → positive delta.
	if dstBoundary != nil {
		totals := e.db.SumCatIDsAtBoundary(dstBoundary.ID)
		for _, m := range parsed.Items {
			if m.Quantity <= 0 {
				continue
			}
			delta := m.Quantity
			qtyAfter := totals[m.CatID]
			qtyBefore := qtyAfter - m.Quantity
			if qtyBefore < 0 {
				qtyBefore = 0
			}
			txns = append(txns, &store.CMSTransaction{
				NodeID:      dstBoundary.ID,
				NodeName:    dstBoundary.Name,
				TxnType:     txnType(delta),
				CatID:       m.CatID,
				Delta:       delta,
				QtyBefore:   qtyBefore,
				QtyAfter:    qtyAfter,
				BinID:       &bin.ID,
				BinLabel:    bin.Label,
				PayloadCode: bin.PayloadCode,
				SourceType:  "movement",
				OrderID:     orderID,
				Notes:       "auto-log",
			})
		}
	}

	if len(txns) == 0 {
		return
	}

	if err := e.db.CreateCMSTransactions(txns); err != nil {
		e.logFn("engine: cms transactions: %v", err)
		return
	}

	e.Events.Emit(Event{Type: EventCMSTransaction, Payload: CMSTransactionEvent{Transactions: txns}})
}

// RecordCorrectionTransactions logs CMS adjustment transactions when a bin's manifest
// is edited. Delta is signed: positive = increase, negative = decrease.
func (e *Engine) RecordCorrectionTransactions(binID, nodeID int64, oldManifest, newManifest []store.ManifestEntry, reason string) {
	boundary := e.FindCMSBoundary(nodeID)
	var boundaryID int64
	var boundaryName string
	if boundary != nil {
		boundaryID = boundary.ID
		boundaryName = boundary.Name
	} else {
		boundaryID = nodeID
		node, err := e.db.GetNode(nodeID)
		if err != nil {
			return
		}
		boundaryName = node.Name
	}

	oldQty := make(map[string]int64)
	for _, m := range oldManifest {
		oldQty[m.CatID] += m.Quantity
	}
	newQty := make(map[string]int64)
	for _, m := range newManifest {
		newQty[m.CatID] += m.Quantity
	}

	bin, err := e.db.GetBin(binID)
	if err != nil {
		return
	}

	var txns []*store.CMSTransaction

	allCatIDs := make(map[string]bool)
	for k := range oldQty {
		allCatIDs[k] = true
	}
	for k := range newQty {
		allCatIDs[k] = true
	}

	totals := e.db.SumCatIDsAtBoundary(boundaryID)
	for catID := range allCatIDs {
		delta := newQty[catID] - oldQty[catID]
		if delta == 0 {
			continue
		}
		qtyAfter := totals[catID]
		qtyBefore := qtyAfter - delta
		if qtyBefore < 0 {
			qtyBefore = 0
		}
		txns = append(txns, &store.CMSTransaction{
			NodeID:      boundaryID,
			NodeName:    boundaryName,
			TxnType:     txnType(delta),
			CatID:       catID,
			Delta:       delta,
			QtyBefore:   qtyBefore,
			QtyAfter:    qtyAfter,
			BinID:       &bin.ID,
			BinLabel:    bin.Label,
			PayloadCode: bin.PayloadCode,
			SourceType:  "correction",
			Notes:       reason,
		})
	}

	if len(txns) == 0 {
		return
	}

	if err := e.db.CreateCMSTransactions(txns); err != nil {
		e.logFn("engine: cms correction transactions: %v", err)
		return
	}

	e.Events.Emit(Event{Type: EventCMSTransaction, Payload: CMSTransactionEvent{Transactions: txns}})
}

