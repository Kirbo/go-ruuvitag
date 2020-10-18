package models

type Config struct {
	Interval      int32       `json:"interval"`
	EnableInserts bool        `json:"enableInserts"`
	EnableRedis   bool        `json:"enableRedis"`
	EnableSocket  bool        `json:"enableSocket"`
	EnableMQTT    bool        `json:"enableMQTT"`
	LogSocket     bool        `json:"logSocket"`
	LogMQTT       bool        `json:"logMQTT"`
	LogInserts    bool        `json:"logInserts"`
	Ruuvitags     JsonDevices `json:"ruuvitags"`
}
