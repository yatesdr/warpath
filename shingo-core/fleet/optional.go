package fleet

// RobotLister provides robot status and control capabilities.
// Web handlers type-assert Backend to this interface.
type RobotLister interface {
	GetRobotsStatus() ([]RobotStatus, error)
	SetAvailability(vehicleID string, available bool) error
	RetryFailed(vehicleID string) error
	ForceComplete(vehicleID string) error
}

// NodeOccupancyProvider provides node location occupancy details.
type NodeOccupancyProvider interface {
	GetNodeOccupancy(groups ...string) ([]OccupancyDetail, error)
}

// VendorProxy exposes the vendor API base URL for raw proxy requests.
type VendorProxy interface {
	BaseURL() string
}

// RobotStatus is a vendor-neutral representation of a robot's state.
type RobotStatus struct {
	VehicleID      string
	Connected      bool
	Available      bool
	Busy           bool
	Emergency      bool
	Blocked        bool
	IsError        bool
	BatteryLevel   float64
	Charging       bool
	CurrentMap     string
	Model          string
	IP             string
	X              float64
	Y              float64
	Angle          float64
	NetworkDelay   int
	CurrentStation string
	LastStation    string
}

// OccupancyDetail is a vendor-neutral representation of a location's occupancy status.
type OccupancyDetail struct {
	ID       string
	Occupied bool
	Holder   int
	Status   int
}
