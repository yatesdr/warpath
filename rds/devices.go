package rds

import (
	"fmt"
	"strings"
)

// CallTerminal sends a command to an external terminal device.
func (c *Client) CallTerminal(req *CallTerminalRequest) (*CallTerminalResponse, error) {
	var resp CallTerminalResponse
	if err := c.post("/callTerminal", req, &resp); err != nil {
		return nil, err
	}
	if err := checkResponse(&resp.Response); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDevicesStatus retrieves status of external devices, optionally filtered by name.
func (c *Client) GetDevicesStatus(devices ...string) (*DevicesResponse, error) {
	path := "/devicesDetails"
	if len(devices) > 0 {
		path = fmt.Sprintf("/devicesDetails?devices=%s", strings.Join(devices, ","))
	}
	var resp DevicesResponse
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	if err := checkResponse(&resp.Response); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CallDoor sends open/close commands to one or more automated doors.
func (c *Client) CallDoor(doors []CallDoorRequest) error {
	var resp Response
	if err := c.post("/callDoor", doors, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// DisableDoor disables or enables automatic door control.
func (c *Client) DisableDoor(req *DisableDeviceRequest) error {
	var resp Response
	if err := c.post("/disableDoor", req, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// CallLift sends commands to one or more lifts/elevators.
func (c *Client) CallLift(lifts []CallLiftRequest) error {
	var resp Response
	if err := c.post("/callLift", lifts, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// DisableLift disables or enables automatic lift control.
func (c *Client) DisableLift(req *DisableDeviceRequest) error {
	var resp Response
	if err := c.post("/disableLift", req, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}
