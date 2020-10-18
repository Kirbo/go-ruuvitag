package models

type Config struct {
	Interval      int32       `json:"interval"`
	EnableInserts bool        `json:"enableInserts"`
	EnableRedis   bool        `json:"enableRedis"`
	EnableSocket  bool        `json:"enableSocket"`
	EnableMQTT    bool        `json:"enableMQTT"`
	Ruuvitags     JsonDevices `json:"ruuvitags"`
}
