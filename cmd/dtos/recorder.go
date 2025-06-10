package dtos

type Recorder struct {
	Name             string   `json:"name"`
	Location         string   `json:"location"`
	VideoDecoder     string   `json:"video_decoder"`
	RTSP             string   `json:"rtsp"`
	StorageProviders []string `json:"storage_providers"`
}
