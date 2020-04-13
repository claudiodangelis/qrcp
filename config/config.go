package config

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path/filepath"

	"github.com/claudiodangelis/qrcp/util"
	"github.com/manifoldco/promptui"
)

// Config of qrcp
type Config struct {
	Interface string `json:"interface"`
	Port      int    `json:"port"`
	KeepAlive bool   `json:"keep-alive"`
}

func configFile() string {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	return filepath.Join(currentUser.HomeDir, ".qrcp.json")
}

// Load a new configuration
func Load() Config {
	var cfg Config
	// Read the configuration file, if it exists
	if file, err := ioutil.ReadFile(configFile()); err == nil {
		// Read the config
		if err := json.Unmarshal(file, &cfg); err != nil {
			panic(err)
		}
	}
	// Prompt if needed
	if cfg.Interface == "" {
		interfacenames, err := util.InterfaceNames()
		if err != nil {
			panic(err)
		}
		if len(interfacenames) == 0 {
			panic("no interfaces found")
		} else if len(interfacenames) > 1 {
			// TODO: Consider showing addresses too
			prompt := promptui.Select{
				Items: interfacenames,
				Label: "Choose interface",
			}
			_, result, err := prompt.Run()
			if err != nil {
				panic(err)
			}
			cfg.Interface = result
		} else {
			cfg.Interface = interfacenames[0]
		}
		// Write config
		// TODO: Implement an .Update method for Config
		j, err := json.Marshal(cfg)
		if err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(configFile(), j, 0644); err != nil {
			panic(err)
		}
	}
	// TODO: Pass port
	return cfg
}
