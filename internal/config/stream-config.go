package config

type StorageProvider interface {
	GetProviderName() string
	UploadFile(filePath string) error
}

type CameraConfig struct {
	Name             string
	Location         string
	RTSP             string
	StorageProviders []StorageProvider
}

type StreamConfig struct {
	StorageProviders []StorageProvider
	Cameras          []CameraConfig
}
