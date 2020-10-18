package models

type Config struct {
	Interval     int32       `json:"interval"`
	Inserts      bool        `json:"inserts"`
	EnableSocket bool        `json:"enableSocket"`
	Ruuvitags    JsonDevices `json:"ruuvitags"`
}
