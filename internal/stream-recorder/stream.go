package stream

import (
	config2 "github.com/gmreis/cam-stream-recorder/internal/config"
)

type StreamService interface {
	Initialize(recorder config2.Recorder, localStoragePath string) error
	StartRecording() error
	StopRecording() error
}
