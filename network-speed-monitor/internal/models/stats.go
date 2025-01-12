package models

type Stat struct {
	Timestamp         string  `json:"timestamp"`
	Connectivity      bool    `json:"connectivity"`
	UploadSpeedMBps   float64 `json:"upload_speed"`
	DownloadSpeedMBps float64 `json:"download_speed"`
}
