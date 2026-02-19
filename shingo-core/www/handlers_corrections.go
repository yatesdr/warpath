package www

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"shingocore/engine"
	"shingocore/store"
)

func (h *Handlers) apiCreateCorrection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CorrectionType string  `json:"correction_type"`
		NodeID         int64   `json:"node_id"`
		PayloadID      int64   `json:"payload_id"`
		CatID          string  `json:"cat_id"`
		Description    string  `json:"description"`
		Quantity       float64 `json:"quantity"`
		Reason         string  `json:"reason"`
		ManifestItemID int64   `json:"manifest_item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	actor := h.getUsername(r)
	if actor == "" {
		actor = "admin"
	}

	corr := &store.Correction{
		CorrectionType: req.CorrectionType,
		NodeID:         req.NodeID,
		PayloadID:      &req.PayloadID,
		CatID:          req.CatID,
		Description:    req.Description,
		Quantity:       req.Quantity,
		Reason:         req.Reason,
		Actor:          actor,
	}

	switch req.CorrectionType {
	case "add_item":
		m := &store.ManifestItem{
			PayloadID:  req.PayloadID,
			PartNumber: req.CatID,
			Quantity:   req.Quantity,
			Notes:      fmt.Sprintf("correction: %s", req.Reason),
		}
		if err := h.engine.DB().CreateManifestItem(m); err != nil {
			h.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		corr.ManifestItemID = &m.ID
	case "remove_item":
		if err := h.engine.DB().DeleteManifestItem(req.ManifestItemID); err != nil {
			h.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		corr.ManifestItemID = &req.ManifestItemID
	case "adjust_qty":
		m := &store.ManifestItem{ID: req.ManifestItemID, Quantity: req.Quantity, PartNumber: req.CatID}
		if err := h.engine.DB().UpdateManifestItem(m); err != nil {
			h.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		corr.ManifestItemID = &req.ManifestItemID
	}

	h.engine.DB().CreateCorrection(corr)

	h.engine.Events.Emit(engine.Event{Type: engine.EventCorrectionApplied, Payload: engine.CorrectionAppliedEvent{
		CorrectionID:   corr.ID,
		CorrectionType: req.CorrectionType,
		NodeID:         req.NodeID,
		Reason:         req.Reason,
		Actor:          actor,
	}})

	h.engine.Events.Emit(engine.Event{Type: engine.EventPayloadChanged, Payload: engine.PayloadChangedEvent{
		NodeID:    req.NodeID,
		Action:    req.CorrectionType,
		PayloadID: req.PayloadID,
	}})

	h.jsonOK(w, map[string]any{"id": corr.ID})
}

func (h *Handlers) apiListNodeCorrections(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
	if err != nil {
		h.jsonError(w, "invalid node_id", http.StatusBadRequest)
		return
	}
	corrections, err := h.engine.DB().ListCorrectionsByNode(nodeID, 20)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.jsonOK(w, corrections)
}
