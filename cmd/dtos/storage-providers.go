package dtos

import (
	"encoding/json"
	"fmt"
)

type StorageProvider interface {
	GetProvider() string
}

type LocalProvider struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
}

func (l LocalProvider) GetProvider() string { return l.Provider }

type AWSProvider struct {
	Provider string `json:"provider"`
	Region   string `json:"region"`
	Bucket   string `json:"bucket"`
	Path     string `json:"path"`
}

func (a AWSProvider) GetProvider() string { return a.Provider }

type storageProviderRaw struct {
	Provider string `json:"provider"`
}

type StorageProviders []StorageProvider

func (s *StorageProviders) UnmarshalJSON(data []byte) error {
	var rawList []json.RawMessage
	if err := json.Unmarshal(data, &rawList); err != nil {
		return err
	}
	for _, raw := range rawList {
		var kind storageProviderRaw
		if err := json.Unmarshal(raw, &kind); err != nil {
			return err
		}
		switch kind.Provider {
		case "local":
			var local LocalProvider
			if err := json.Unmarshal(raw, &local); err != nil {
				return err
			}
			*s = append(*s, local)
		case "aws":
			var aws AWSProvider
			if err := json.Unmarshal(raw, &aws); err != nil {
				return err
			}
			*s = append(*s, aws)
		default:
			return fmt.Errorf("provider unknown: %s", kind.Provider)
		}
	}
	return nil
}
