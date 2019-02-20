package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

// Config holds the values
type Config struct {
	Iface string `json:"interface"`
	Port  int    `json:"port"`
}

func configFile() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(currentUser.HomeDir, ".qr-filetransfer.json"), nil
}

// Update the configuration file
func (c *Config) Update() error {
	debug("Updating config file")
	j, err := json.Marshal(c)
	if err != nil {
		return err
	}
	file, err := configFile()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, j, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Delete the configuration file
func (c *Config) Delete() (bool, error) {
	file, err := configFile()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false, nil
	}
	if err := os.Remove(file); err != nil {
		return false, err
	}
	return true, nil
}

// LoadConfig from file
func LoadConfig() Config {
	var cfg Config
	file, err := configFile()
	if err != nil {
		return cfg
	}
	debug("Current config file is", file)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return cfg
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		log.Println("WARN:", err)
	}
	return cfg
}
