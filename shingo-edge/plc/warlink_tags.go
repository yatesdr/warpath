package plc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// WarlinkTagInfo describes a tag from the WarLink all-tags endpoint,
// including tags that are not yet enabled for REST publishing.
type WarlinkTagInfo struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Configured bool        `json:"configured"`
	Enabled    bool        `json:"enabled"`
	Writable   bool        `json:"writable,omitempty"`
	NoREST     bool        `json:"no_rest,omitempty"`
	Value      interface{} `json:"value,omitempty"`
}

// FetchAllTags retrieves ALL tags (published and unpublished) from WarLink
// via GET /api/{plcName}/all-tags.
func (m *Manager) FetchAllTags(ctx context.Context, plcName string) ([]WarlinkTagInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.baseURL()+"/"+plcName+"/all-tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WarLink all-tags %s returned %d", plcName, resp.StatusCode)
	}
	var tags []WarlinkTagInfo
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode all-tags %s: %w", plcName, err)
	}
	return tags, nil
}

// EnableTagPublishing tells WarLink to start publishing a tag via
// PATCH /api/{plcName}/tags/{tagName} with {"enabled":true,"no_rest":false}.
func (m *Manager) EnableTagPublishing(ctx context.Context, plcName, tagName string) error {
	body, _ := json.Marshal(map[string]interface{}{"enabled": true, "no_rest": false})
	return m.patchTag(ctx, plcName, tagName, body)
}

// DisableTagPublishing tells WarLink to stop publishing a tag via
// PATCH /api/{plcName}/tags/{tagName} with {"enabled":false}.
func (m *Manager) DisableTagPublishing(ctx context.Context, plcName, tagName string) error {
	body, _ := json.Marshal(map[string]interface{}{"enabled": false})
	return m.patchTag(ctx, plcName, tagName, body)
}

func (m *Manager) patchTag(ctx context.Context, plcName, tagName string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "PATCH", m.baseURL()+"/"+plcName+"/tags/"+tagName, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("WarLink PATCH %s/%s returned %d", plcName, tagName, resp.StatusCode)
	}
	return nil
}

// IsTagPublished checks whether a tag is currently in the local WarLink cache
// (i.e. it's already being published and polled).
func (m *Manager) IsTagPublished(plcName, tagName string) bool {
	m.mu.RLock()
	mp, ok := m.plcs[plcName]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	_, exists := mp.Values[tagName]
	return exists
}
