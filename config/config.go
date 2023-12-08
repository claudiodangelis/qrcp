package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/asaskevich/govalidator"
	"github.com/claudiodangelis/qrcp/application"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/viper"
)

type Config struct {
	Interface string `yaml:",omitempty"`
	Port      int    `yaml:",omitempty"`
	Bind      string `yaml:",omitempty"`
	KeepAlive bool   `yaml:",omitempty"`
	Path      string `yaml:",omitempty"`
	Secure    bool   `yaml:",omitempty"`
	TlsKey    string `yaml:",omitempty"`
	TlsCert   string `yaml:",omitempty"`
	FQDN      string `yaml:",omitempty"`
	Output    string `yaml:",omitempty"`
	Reversed  bool   `yaml:",omitempty"`
}

var interactive bool = false

func New(app application.App) Config {
	v := getViperInstance(app)
	var err error
	cfg := Config{}

	_, err = os.Stat(v.ConfigFileUsed())
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(v.ConfigFileUsed()), os.ModeDir|os.ModePerm); err != nil {
			panic(err)
		}
		file, err := os.Create(v.ConfigFileUsed())
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	// Load file
	cfg.Interface = v.GetString("interface")
	cfg.Bind = v.GetString("bind")
	cfg.Port = v.GetInt("port")
	cfg.KeepAlive = v.GetBool("keepAlive")
	cfg.Path = v.GetString("path")
	cfg.Secure = v.GetBool("secure")
	cfg.TlsKey = v.GetString("tls-key")
	cfg.TlsCert = v.GetString("tls-cert")
	cfg.FQDN = v.GetString("fqdn")
	cfg.Output = v.GetString("output")
	cfg.Reversed = v.GetBool("reversed")

	// Override
	if app.Flags.Interface != "" {
		cfg.Interface = app.Flags.Interface
	}
	if app.Flags.Bind != "" {
		cfg.Bind = app.Flags.Bind
	}
	if app.Flags.Port != 0 {
		cfg.Port = app.Flags.Port
	}
	if app.Flags.KeepAlive {
		cfg.KeepAlive = true
	}
	if app.Flags.Path != "" {
		cfg.Path = app.Flags.Path
	}
	if app.Flags.Secure {
		cfg.Secure = true
	}
	if app.Flags.TlsKey != "" {
		cfg.TlsKey = app.Flags.TlsKey
	}
	if app.Flags.TlsCert != "" {
		cfg.TlsCert = app.Flags.TlsCert
	}
	if app.Flags.FQDN != "" {
		cfg.FQDN = app.Flags.FQDN
	}
	if app.Flags.Output != "" {
		cfg.Output = app.Flags.Output
	}
	if app.Flags.Reversed {
		cfg.Reversed = true
	}

	// Discover interface if it's not been set yet
	if !interactive {
		if cfg.Interface == "" {
			cfg.Interface, err = chooseInterface(app.Flags)
			if err != nil {
				panic(err)
			}
			v.Set("interface", cfg.Interface)
			if err := v.WriteConfig(); err != nil {
				panic(err)
			}
		}
	}

	return cfg
}

func getViperInstance(app application.App) *viper.Viper {
	var configType string
	var configFile string
	v := viper.New()
	if app.Flags.Config != "" {
		configFile = app.Flags.Config
		configType = filepath.Ext(configFile)[1:]
	} else {
		oldConfigFile := filepath.Join(xdg.ConfigHome, "qrcp", "config.json")
		// Check if old configuration file exists
		if _, err := os.Stat(oldConfigFile); os.IsNotExist(err) {
			configType = "yml"
		} else {
			configType = "json"
		}
		configFile = filepath.Join(xdg.ConfigHome, app.Name, fmt.Sprintf("config.%s", configType))
	}
	v.SetConfigType(configType)
	v.SetConfigFile(configFile)
	v.AutomaticEnv()
	v.SetEnvPrefix(app.Name)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	return v
}

func Wizard(app application.App) error {
	interactive = true
	cfg := New(app)
	v := getViperInstance(app)
	// Choose interface
	var err error
	cfg.Interface, err = chooseInterface(app.Flags)
	if err != nil {
		panic(err)
	}
	v.Set("interface", cfg.Interface)
	if err := v.WriteConfig(); err != nil {
		panic(err)
	}
	// Ask for bind address
	validateBind := func(input string) error {
		if input == "" {
			return nil
		}
		if !govalidator.IsIPv4(input) {
			return errors.New("invalid address")
		}
		return nil
	}
	promptBind := promptui.Prompt{
		Validate: validateBind,
		Label:    "Enter bind address (this will override the chosen interface address)",
		Default:  cfg.Bind,
	}
	if promptBindResultString, err := promptBind.Run(); err == nil {
		if promptBindResultString != "" {
			v.Set("bind", promptBindResultString)
		}
	}
	// Ask for port
	validatePort := func(input string) error {
		_, err := strconv.ParseUint(input, 10, 16)
		if err != nil {
			return errors.New("invalid number")
		}
		return nil
	}
	promptPort := promptui.Prompt{
		Validate: validatePort,
		Label:    "Choose port, 0 means random port",
		Default:  fmt.Sprintf("%d", cfg.Port),
	}
	if promptPortResultString, err := promptPort.Run(); err == nil {
		if port, err := strconv.ParseUint(promptPortResultString, 10, 16); err == nil {
			if port > 0 {
				v.Set("port", port)
			}
		}
	}
	// Ask for fully qualified domain name
	validateFqdn := func(input string) error {
		if input != "" && !govalidator.IsDNSName(input) {
			return errors.New("invalid domain")
		}
		return nil
	}
	promptFqdn := promptui.Prompt{
		Validate: validateFqdn,
		Label:    "Choose fully-qualified domain name",
		Default:  cfg.FQDN,
	}
	if promptFqdnString, err := promptFqdn.Run(); err == nil {
		if promptFqdnString != "" {
			v.Set("fqdn", promptFqdnString)
		}

	}
	promptPath := promptui.Prompt{
		Label:   "Choose URL path, empty means random",
		Default: cfg.Path,
	}
	if promptPathResultString, err := promptPath.Run(); err == nil {
		if promptPathResultString != "" {
			v.Set("path", promptPathResultString)
		}
	}
	// Ask for keep alive
	promptKeepAlive := promptui.Select{
		Items: []string{"No", "Yes"},
		Label: "Should the server keep alive after transferring?",
	}
	if _, promptKeepAliveResultString, err := promptKeepAlive.Run(); err == nil {
		if promptKeepAliveResultString == "Yes" {
			v.Set("keepAlive", true)
		}
	}
	// HTTPS
	// Ask if path is readable and is a file
	pathIsReadableFile := func(input string) error {
		if input == "" {
			return errors.New("invalid path")
		}
		path, err := filepath.Abs(util.Expand(input))
		if err != nil {
			return err
		}
		fmt.Println(path)
		fileinfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fileinfo.Mode().IsDir() {
			return fmt.Errorf(fmt.Sprintf("%s is a directory", input))
		}
		return nil
	}
	promptSecure := promptui.Select{
		Items: []string{"No", "Yes"},
		Label: "Should files be securely transferred with HTTPS?",
	}
	if _, promptSecureResultString, err := promptSecure.Run(); err == nil {
		if promptSecureResultString == "Yes" {
			v.Set("secure", true)
		}
		cfg.Secure = v.GetBool("secure")
	}
	if cfg.Secure {
		// TLS Cert
		promptTlsCert := promptui.Prompt{
			Label:    "Choose TLS certificate path. Empty if not using HTTPS.",
			Default:  cfg.TlsCert,
			Validate: pathIsReadableFile,
		}
		if promptTlsCertString, err := promptTlsCert.Run(); err == nil {
			v.Set("tlsCert", util.Expand(promptTlsCertString))
		}
		// TLS key
		promptTlsKey := promptui.Prompt{
			Label:    "Choose TLS certificate key. Empty if not using HTTPS.",
			Default:  cfg.TlsKey,
			Validate: pathIsReadableFile,
		}
		if promptTlsKeyString, err := promptTlsKey.Run(); err == nil {
			v.Set("tlsKey", util.Expand(promptTlsKeyString))
		}
	}
	validateIsDir := func(input string) error {
		if input == "" {
			return nil
		}
		path, err := filepath.Abs(input)
		if err != nil {
			return err
		}
		f, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !f.IsDir() {
			return errors.New("path is not a directory")
		}
		return nil
	}
	// Ask for default output directory
	promptOutput := promptui.Prompt{
		Label:    "Choose default output directory for received files, empty does not set a default",
		Default:  cfg.Output,
		Validate: validateIsDir,
	}
	if promptOutputResultString, err := promptOutput.Run(); err == nil {
		if promptOutputResultString != "" {
			output, _ := filepath.Abs(promptOutputResultString)
			v.Set("output", output)
		}
	}
	promptReversed := promptui.Select{
		Items: []string{"No", "Yes"},
		Label: "Reverse QR code (black text on white background)?",
	}
	if _, promptReversedResultString, err := promptReversed.Run(); err == nil {
		if promptReversedResultString == "Yes" {
			v.Set("reversed", true)
		}
		cfg.Reversed = v.GetBool("reversed")
	}

	return v.WriteConfig()
}
