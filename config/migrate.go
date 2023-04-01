package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/claudiodangelis/qrcp/application"
	"gopkg.in/yaml.v2"
)

// Migrate function will look for an existing legacy configuration file
// and will migrate it to the new format
func Migrate(app application.App) (bool, error) {
	oldConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.json")
	newConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.yml")
	// Check if old configuration file exists
	if _, err := os.Stat(oldConfigFile); os.IsNotExist(err) {
		return false, nil
	}
	oldConfigFileBytes, err := ioutil.ReadFile(oldConfigFile)
	if err != nil {
		panic(err)
	}
	var cfg Config
	if err := json.Unmarshal(oldConfigFileBytes, &cfg); err != nil {
		panic(err)
	}
	newConfigFileBytes, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(newConfigFile, newConfigFileBytes, 0644); err != nil {
		panic(err)
	}
	// Delete old file
	if err := os.Remove(oldConfigFile); err != nil {
		panic(err)
	}
	return true, nil
}
