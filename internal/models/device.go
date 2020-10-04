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
	OldID        string
	Name         string
	Ping         int64
	Format       uint8
	Humidity     float32
	Temperature  float32
	Pressure     float32
	Timestamp    int64
	TimestampZ   string
	Acceleration DeviceAcceleration
	Battery      float32
}

type BroadcastMessage struct {
	Timestamp   string  `json:"time"`
	TagID       string  `json:"tagId"`
	Name        string  `json:"name"`
	Temperature float32 `json:"temperature"`
	Pressure    float32 `json:"pressure"`
	Battery     float32 `json:"battery"`
	Humidity    float32 `json:"humidity"`
	Ping        int64   `json:"ping"`
}
