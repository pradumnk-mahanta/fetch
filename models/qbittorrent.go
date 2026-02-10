package models

type QBitTorrent struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
	Dlspeed  int     `json:"dlspeed"`
	Upspeed  int     `json:"upspeed"`
	State    string  `json:"state"` // e.g., "downloading", "pausedDL"
	Category string  `json:"category"`
	Eta      int     `json:"eta"`
}
