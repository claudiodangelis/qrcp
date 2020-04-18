package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/manifoldco/promptui"
)

// Config of qrcp
type Config struct {
	FQDN      string `json:"fqdn"`
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

type chooseInterfaceOptions struct {
	interactive bool
}

func chooseInterface(opts chooseInterfaceOptions) (string, error) {
	interfaces, err := util.Interfaces()
	if err != nil {
		return "", err
	}
	if len(interfaces) == 0 {
		return "", errors.New("no interfaces found")
	}

	if len(interfaces) == 1 && opts.interactive == false {
		for name := range interfaces {
			fmt.Printf("only one interface found: %s, using this one\n", name)
			return name, nil
		}
	}
	// Map for pretty printing
	m := make(map[string]string)
	items := []string{}
	for name, ip := range interfaces {
		label := fmt.Sprintf("%s (%s)", name, ip)
		m[label] = name
		items = append(items, label)
	}
	// Add the "any" interface
	anyIP := "0.0.0.0"
	anyName := "any"
	anyLabel := fmt.Sprintf("%s (%s)", anyName, anyIP)
	m[anyLabel] = anyName
	items = append(items, anyLabel)
	prompt := promptui.Select{
		Items: items,
		Label: "Choose interface",
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return m[result], nil
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
		iface, err := chooseInterface(chooseInterfaceOptions{})
		if err != nil {
			log.Fatalln(err)
		}
		cfg.Interface = iface
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
	opts := chooseInterfaceOptions{
		interactive: true,
	}
	iface, err := chooseInterface(opts)
	if err != nil {
		log.Fatalln(err)
	}
	cfg.Interface = iface
	// Ask for fully qualified domain name
	validateFqdn := func(input string) error {
		if input != "" && govalidator.IsDNSName(input) == false {
			return errors.New("invalid domain")
		}
		return nil
	}
	promptFqdn := promptui.Prompt{
		Validate: validateFqdn,
		Label:    "Choose fully-qualified domain name",
		Default:  "",
	}
	if promptFqdnString, err := promptFqdn.Run(); err == nil {
		cfg.FQDN = promptFqdnString
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
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Configuration updated:\n%s\n", string(b))
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
func New(iface string, port int, fqdn string, keepAlive bool) Config {
	// Load saved file / defults
	cfg := Load()
	if iface != "" {
		cfg.Interface = iface
	}
	if fqdn != "" {
		if govalidator.IsDNSName(fqdn) == false {
			panic("invalid value for fully-qualified domain name")
		}
		cfg.FQDN = fqdn
	}
	if port != 0 {
		if port > 65535 {
			panic("invalid value for port")
		}
		cfg.Port = port
	}
	if keepAlive {
		cfg.KeepAlive = true
	}

	return cfg
}
