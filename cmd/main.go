package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gmreis/cam-stream-recorder/cmd/dtos"
	"github.com/gmreis/cam-stream-recorder/internal/config"
	"github.com/gmreis/cam-stream-recorder/internal/stream-recorder"
)

func loadConfig(path string) (*dtos.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config:\n\r %w", err)
	}
	var cfg dtos.Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error in parsing config:\n\r %w", err)
	}
	return &cfg, nil
}

func startStreamWorker(ctx context.Context, wg *sync.WaitGroup, recorder dtos.Recorder, defaultConfig dtos.DefaultConfig) {
	defer wg.Done()

	// Convert dtos.Recorder to config.CameraConfig
	cameraConfig := config.Recorder{
		Name:             recorder.Name,
		Location:         recorder.Location,
		RTSP:             recorder.RTSP,
		StorageProviders: []config.StorageProvider{}, // recorder.StorageProviders, // Provider configurations
	}

	streamRecorder := stream.NewStreamRecorder()
	err := streamRecorder.Initialize(cameraConfig, "LocalStoragePath")
	if err != nil {
		log.Printf("Error initializing stream recorder for camera %s: %v", recorder.Name, err)
		return
	}

	done := make(chan struct{})
	go func() {
		err = streamRecorder.StartRecording()
		if err != nil {
			log.Printf("Error starting recording for camera %s: %v", recorder.Name, err)
			return
		}
		close(done)
	}()

	log.Printf("Recording started for camera %s", recorder.Name)

	// Wait for the context to be done or for the recording to finish
	select {
	case <-ctx.Done(): // TODO: When ctx is cancelled, the StopRecording will be called???
		streamRecorder.StopRecording()
	case <-done:
		streamRecorder.StopRecording()
	}

	log.Printf("Recording stopped for camera %s", recorder.Name)
}

func shutdownHandler(wg *sync.WaitGroup, cancel context.CancelFunc, stop chan os.Signal) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	log.Println("Application is running... Press Ctrl+C to exit.")

	select {
	case <-stop:
		log.Println("Received stop signal, stopping recording...")
		cancel()
		<-done
	case <-done:
		log.Println("All recordings finished.")
	}

	log.Println("All recordings stopped, cleaning up...")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Use: ./cam-stream-recorder <caminho_config.json>")
		os.Exit(1)
	}
	configPath := os.Args[1]
	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Println("Fail to load config:", err)
		os.Exit(1)
	}
	fmt.Printf("Config loaded successfully: %+v\n", cfg)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	for _, recorder := range cfg.Recorders {
		wg.Add(1)

		go startStreamWorker(ctx, wg, recorder, cfg.DefaultConfig)
	}

	shutdownHandler(wg, cancel, stop)
}
