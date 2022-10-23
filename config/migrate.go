package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// migrate function will look for an existing legacy configuration file
// and will migrate it to the new format
func migrate() {
	// Return if old file does not exist
	oldConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.json")
	newConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.json")

	if _, err := os.Stat(oldConfigFile); os.IsNotExist(err) {
		fmt.Println("old config file does not exist, skipping")
		return
	}
	// Delete old file if new file exists and return
	if _, err := os.Stat(newConfigFile); !os.IsNotExist(err) {
		fmt.Println("new config file exists, delete old")
		if err := os.Remove(oldConfigFile); err != nil {
			fmt.Println("warning: error while deleting old configuration file", err)
		}
		return
	}
	// Migrate content
	oldConfig, err := ioutil.ReadFile(oldConfigFile)
	if err != nil {
		fmt.Println("warning: error while reading contents of old configuration file", err)
		return
	}
	var oldConfigStruct interface{}
	if err := json.Unmarshal(oldConfig, &oldConfigStruct); err != nil {
		fmt.Println("warning: error while parsing JSON from old configuration file", err)
		return
	}
	fmt.Printf("%+v\n", oldConfigStruct)
}
