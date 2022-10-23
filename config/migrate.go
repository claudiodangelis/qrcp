package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v2"
)

// migrate function will look for an existing legacy configuration file
// and will migrate it to the new format
func migrate() {
	// Return if old file does not exist
	oldConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.json")
	newConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.yml")

	if _, err := os.Stat(oldConfigFile); os.IsNotExist(err) {
		return
	}
	// Delete old file if new file exists and return
	if _, err := os.Stat(newConfigFile); !os.IsNotExist(err) {
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
	var oldConfigStruct struct {
		Fqdn      string `yaml:"fqdn,omitempty"`
		Interface string `yaml:"interface,omitempty"`
		Keepalive bool   `yaml:"keepalive,omitempty"`
		Output    string `yaml:"output,omitempty"`
		Path      string `yaml:"path,omitempty"`
		Port      int    `yaml:"port,omitempty"`
		Secure    bool   `yaml:"secure,omitempty"`
		Tlscert   string `yaml:"tls-cert,omitempty"`
		Tlskey    string `yaml:"tls-key,omitempty"`
	}
	if err := yaml.Unmarshal(oldConfig, &oldConfigStruct); err != nil {
		fmt.Println("warning: error while parsing JSON from old configuration file", err)
		return
	}
	newConfig, err := yaml.Marshal(oldConfigStruct)
	if err != nil {
		fmt.Println("warning: error while migrating JSON file to YAML file", err)
	}
	// Heads-up: Replace tls-cert, tls-key with tlscert, tlskey
	if err := os.WriteFile(newConfigFile, []byte(strings.ReplaceAll(string(newConfig), "tls-", "tls")), 0644); err != nil {
		fmt.Println("warning: error while creating the migrated YAML file", err)
	}
	// Delete old file
	if err := os.Remove(oldConfigFile); err != nil {
		fmt.Println("warning: error while deleting old configuration file", err)
	}
}
