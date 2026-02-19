package rds

import "io"

// DownloadScene downloads the complete scene configuration as raw binary.
func (c *Client) DownloadScene() ([]byte, error) {
	return c.getRaw("/downloadScene")
}

// UploadScene uploads a new scene configuration from raw binary data.
func (c *Client) UploadScene(data io.Reader) error {
	var resp Response
	if err := c.postRaw("/uploadScene", "application/octet-stream", data, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// SyncScene pushes the current scene configuration to all connected robots.
func (c *Client) SyncScene() error {
	var resp Response
	if err := c.post("/syncScene", nil, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}
