package config

type StorageProvider interface {
	GetProviderName() string
	UploadFile(filePath string) error
}

type Recorder struct {
	Name             string
	Location         string
	RTSP             string
	StorageProviders []StorageProvider
}

type StreamConfig struct {
	StorageProviders []StorageProvider
	Recorders        []Recorder
}
