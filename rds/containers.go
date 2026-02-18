package rds

// BindContainerGoods associates a goods ID with a container on a robot.
func (c *Client) BindContainerGoods(req *BindGoodsRequest) error {
	var resp Response
	if err := c.post("/setContainerGoods", req, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// UnbindGoods removes a goods binding by vehicle and goods ID.
func (c *Client) UnbindGoods(vehicle, goodsID string) error {
	var resp Response
	if err := c.post("/clearGoods", &UnbindGoodsRequest{Vehicle: vehicle, GoodsID: goodsID}, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// UnbindContainerGoods removes all goods from a specific container on a robot.
func (c *Client) UnbindContainerGoods(vehicle, containerName string) error {
	var resp Response
	if err := c.post("/clearContainer", &UnbindContainerRequest{Vehicle: vehicle, ContainerName: containerName}, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}

// ClearAllContainerGoods removes all goods from all containers on a robot.
func (c *Client) ClearAllContainerGoods(vehicle string) error {
	var resp Response
	if err := c.post("/clearAllContainersGoods", &ClearAllGoodsRequest{Vehicle: vehicle}, &resp); err != nil {
		return err
	}
	return checkResponse(&resp)
}
