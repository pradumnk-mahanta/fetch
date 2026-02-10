package models

type SabQueueResponse struct {
	Queue SabQueueData `json:"queue"`
}

type SabQueueData struct {
	Status   string    `json:"status"`
	Speed    string    `json:"speed"`
	Size     string    `json:"size"`
	SizeLeft string    `json:"sizeleft"`
	Version  string    `json:"version"`
	Slots    []SabSlot `json:"slots"`
}

type SabSlot struct {
	NzoID    string `json:"nzo_id"`
	Filename string `json:"filename"`
	Size     string `json:"size"`
	Status   string `json:"status"`
	Index    int    `json:"index"`
}

type SabAddResponse struct {
	Status bool     `json:"status"`
	NzoIDs []string `json:"nzo_ids"`
}
