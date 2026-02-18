package rds

import "encoding/json"

// GetSimStateTemplate retrieves the template for simulated robot state fields.
func (c *Client) GetSimStateTemplate() (json.RawMessage, error) {
	var resp SimStateTemplateResponse
	if err := c.get("/getSimRobotStateTemplate", &resp); err != nil {
		return nil, err
	}
	if err := checkResponse(&resp.Response); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// UpdateSimState sets the state of a simulated robot.
func (c *Client) UpdateSimState(req map[string]any) error {
	var resp Response
	if err := c.post("/updateSimRobotState", req, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}
