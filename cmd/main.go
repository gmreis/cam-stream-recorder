package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gmreis/cam-stream-recorder/cmd/dtos"
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

	// ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// defer stop()

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Println("Recovered from panic:", r)
	// 	}
	// }()

	fmt.Println("Application is running... Press Ctrl+C to exit.")

}
