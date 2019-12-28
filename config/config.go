package config

import (
	"os/user"
	"path/filepath"
)

// Config of qrcp
type Config struct {
	Interface string `json:"interface"`
	Port      int    `json:"port"`
}

func configFile() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(currentUser.HomeDir, ".qr-filetransfer.json"), nil
}

// Load a new configuration
func Load() Config {
	var cfg Config
	// Read the file
	// If it's empty
	// Prompt if needed
	return cfg
}
