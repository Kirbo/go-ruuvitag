package models

type Config struct {
	Interval               int32       `json:"interval"`
	SocketInterval         int32       `json:"socketInterval"`
	EnableInserts          bool        `json:"enableInserts"`
	EnableRedis            bool        `json:"enableRedis"`
	EnableSocket           bool        `json:"enableSocket"`
	EnableMQTT             bool        `json:"enableMQTT"`
	LogSocket              bool        `json:"logSocket"`
	LogMQTT                bool        `json:"logMQTT"`
	LogInserts             bool        `json:"logInserts"`
	LogReloadNames         bool        `json:"logReloadNames"`
	PushSocketImmediatelly bool        `json:"pushSocketImmediatelly"`
	PushSocketOnIntervals  bool        `json:"pushSocketOnIntervals"`
	Ruuvitags              JsonDevices `json:"ruuvitags"`
}
