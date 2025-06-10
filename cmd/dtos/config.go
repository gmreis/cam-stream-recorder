package dtos

type Config struct {
	Recorders          []Recorder       `json:"recorders"`
	StorageProviders   StorageProviders `json:"storage_providers"`
	LocalStoragePath   string           `json:"local_storage_path"`
	MaxSizeInMegabytes int              `json:"max_size_in_megabytes"`
}
