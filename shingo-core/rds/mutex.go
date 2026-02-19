package rds

import "encoding/json"

// OccupyMutexGroup claims exclusive access to mutex groups. Returns bare JSON array.
func (c *Client) OccupyMutexGroup(id string, groups []string) ([]MutexGroupResult, error) {
	var results []MutexGroupResult
	if err := c.post("/getBlockGroup", &MutexGroupRequest{ID: id, BlockGroup: groups}, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ReleaseMutexGroup releases previously claimed mutex groups. Returns bare JSON array.
func (c *Client) ReleaseMutexGroup(id string, groups []string) ([]MutexGroupResult, error) {
	var results []MutexGroupResult
	if err := c.post("/releaseBlockGroup", &MutexGroupRequest{ID: id, BlockGroup: groups}, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetMutexGroupStatus retrieves the status of mutex groups. Returns bare JSON array.
func (c *Client) GetMutexGroupStatus(groups []string) ([]MutexGroupStatus, error) {
	data, err := c.getRaw("/blockGroupStatus")
	if err != nil {
		return nil, err
	}
	var results []MutexGroupStatus
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}
