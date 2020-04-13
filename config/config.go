package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/claudiodangelis/qrcp/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/pflag"
)

// Config of qrcp
type Config struct {
	Interface string `json:"interface"`
	Port      int    `json:"port"`
	KeepAlive bool   `json:"keepAlive"`
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
		if err := write(cfg); err != nil {
			panic(err)
		}
	}
	return cfg
}

// Wizard starts an interactive configuration managements
func Wizard() error {
	var cfg Config
	if file, err := ioutil.ReadFile(configFile()); err == nil {
		// Read the config
		if err := json.Unmarshal(file, &cfg); err != nil {
			panic(err)
		}
	}
	// Ask for interface
	interfacenames, err := util.InterfaceNames()
	if err != nil {
		panic(err)
	}
	if len(interfacenames) == 0 {
		panic("no interfaces found")
	} else {
		// TODO: Consider showing addresses too
		promptInterface := promptui.Select{
			Items: interfacenames,
			Label: "Choose interface",
		}
		_, result, err := promptInterface.Run()
		if err != nil {
			panic(err)
		}
		cfg.Interface = result
	}
	// Ask for port
	validatePort := func(input string) error {
		_, err := strconv.ParseInt(input, 10, 16)
		if err != nil {
			return errors.New("Invalid number")
		}
		return nil
	}

	promptPort := promptui.Prompt{
		Validate: validatePort,
		Label:    "Choose port, 0 means random port",
		Default:  fmt.Sprintf("%d", cfg.Port),
	}
	// TODO: Rename this variable maybe?
	if promptPortResultString, err := promptPort.Run(); err == nil {
		if port, err := strconv.ParseInt(promptPortResultString, 10, 16); err == nil {
			cfg.Port = int(port)
		}
	}
	// Ask for keep alive
	promptKeepAlive := promptui.Select{
		Items: []string{"No", "Yes"},
		Label: "Should the server keep alive after transfering?",
	}
	if _, promptKeepAliveResultString, err := promptKeepAlive.Run(); err == nil {
		if promptKeepAliveResultString == "Yes" {
			cfg.KeepAlive = true
		} else {
			cfg.KeepAlive = false
		}
	}
	// Write it down
	if err := write(cfg); err != nil {
		return err
	}
	return nil
}

// write the configuration file to disk
func write(cfg Config) error {
	j, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(configFile(), j, 0644); err != nil {
		return err
	}
	return nil
}

// New returns a new configuration struct. It loads defaults, then overrides
// values if any.
func New(flags *pflag.FlagSet) Config {
	// Load saved file / defults
	cfg := Load()
	// TODO: It looks like there is room for improvement here
	if iface, _ := flags.GetString("interface"); iface != "" {
		cfg.Interface = iface
	}
	if port, _ := flags.GetInt("port"); port != 0 {
		cfg.Port = port
	}
	if keepAlive, _ := flags.GetBool("keep-alive"); keepAlive {
		cfg.KeepAlive = true
	}
	return cfg
}
