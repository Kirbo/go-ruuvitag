package models

type MQTTUser struct {
	CliendID string `json:"clientId"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type EnabledMetrics struct {
	Battery     bool `json:"battery"`
	Humidity    bool `json:"humidity"`
	Pressure    bool `json:"pressure"`
	Temperature bool `json:"temperature"`
	X           bool `json:"x"`
	Y           bool `json:"y"`
	Z           bool `json:"z"`
}

type MQTTConfig struct {
	Host           string         `json:"mqtthost"`
	Port           int32          `json:"mqttport"`
	User           MQTTUser       `json:"mqttuser"`
	EnabledMetrics EnabledMetrics `json:"enabledmetrics"`
}
