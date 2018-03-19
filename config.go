package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/user"
)

// Config holds the values
type Config struct {
	Iface string `json:"interface"`
}

// Update the configuration file
func (c *Config) Update() error {
	debug("Updating config file")
	j, err := json.Marshal(c)
	if err != nil {
		return err
	}
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(currentUser.HomeDir+"/.qr-filetransfer.json", j, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Delete the configuration file
func (c *Config) Delete() (bool, error) {
	currentUser, err := user.Current()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(currentUser.HomeDir + "/.qr-filetransfer.json"); os.IsNotExist(err) {
		return false, nil
	}
	if err := os.Remove(currentUser.HomeDir + "/.qr-filetransfer.json"); err != nil {
		return false, err
	}
	return true, nil
}

// LoadConfig from file
func LoadConfig() Config {
	var config Config
	currentUser, err := user.Current()
	if err != nil {
		return config
	}
	debug("Current user is", currentUser.HomeDir)
	configFile, err := ioutil.ReadFile(currentUser.HomeDir + "/.qr-filetransfer.json")
	if err != nil {
		return config
	}
	if err = json.Unmarshal(configFile, &config); err != nil {
		log.Println("WARN:", err)
	}
	return config
}
