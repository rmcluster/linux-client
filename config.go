package main

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ID string `json:"id"`
}

func getConfigPath() string {
	return filepath.Join(getDataPath(), "config.json")
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(config *Config) error {
	configPath := getConfigPath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, configData, 0644)
}

func GenerateRandomID() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = charset[int(bytes[i])%len(charset)]
	}
	return string(bytes), nil
}

func LoadOrCreateConfig() (*Config, error) {
	config, err := LoadConfig()
	if err == nil && config.ID != "" {
		return config, nil
	}

	// Generate new ID
	id, err := GenerateRandomID()
	if err != nil {
		return nil, err
	}

	config = &Config{ID: id}

	if err := SaveConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}
