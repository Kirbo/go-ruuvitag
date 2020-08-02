package models

type JsonDevice struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}
type JsonDevices []JsonDevice

type DeviceAcceleration struct {
	X float32
	Y float32
	Z float32
}

// Device type
type Device struct {
	ID           string
	Name         string
	Ping         int64
	Format       uint8
	Humidity     float32
	Temperature  float32
	Pressure     uint32
	Timestamp    int64
	Acceleration DeviceAcceleration
	Battery      float32
}
