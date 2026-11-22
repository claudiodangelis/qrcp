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
	"github.com/claudiodangelis/qrcp/logger"
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
	Scheme    string `yaml:",omitempty"`
	WithName  bool   `yaml:",omitempty"`
}

var interactive bool = false

func New(app application.App) Config {
	log := logger.New(app.Flags.Quiet)
	v := getViperInstance(app)
	_, err := os.Stat(v.ConfigFileUsed())
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(v.ConfigFileUsed()), os.ModeDir|os.ModePerm); err != nil {
			panic(err)
		}
		file, err := os.Create(v.ConfigFileUsed())
		if err != nil {
			panic(err)
		}
		if err := file.Close(); err != nil {
			panic(err)
		}
	}
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	cfg := fromViper(v)
	applyFlags(&cfg, app.Flags)

	// Discover interface if it's not been set yet
	if !interactive {
		if cfg.Interface == "" {
			cfg.Interface, err = chooseInterface(app.Flags)
			if err != nil {
				panic(err)
			}
			v.Set("interface", cfg.Interface)
			if err := v.WriteConfig(); err != nil {
				log.Print(fmt.Sprintf("Warning: the configuration file could not be saved: %v\n", err))
			}
		}
	}

	return cfg
}

// Load reads the configuration file, if it exists, and applies the flags and
// environment variables. Unlike New, it never creates the file, chooses an
// interface or writes anything, and it returns errors instead of panicking.
// If the file can't be read, the error is returned with the configuration
// from the flags and environment variables only.
func Load(app application.App) (Config, error) {
	if app.Flags.Config != "" && filepath.Ext(app.Flags.Config) == "" {
		// getViperInstance takes the format from the extension, so read the
		// environment variables only
		v := viper.New()
		readEnv(v, app)
		cfg := fromViper(v)
		applyFlags(&cfg, app.Flags)
		return cfg, fmt.Errorf("the configuration file %s has no extension, such as .yml, to tell its format", app.Flags.Config)
	}
	v := getViperInstance(app)
	var err error
	if _, statErr := os.Stat(v.ConfigFileUsed()); statErr == nil {
		if readErr := v.ReadInConfig(); readErr != nil {
			err = fmt.Errorf("unable to read the configuration file %s: %w", v.ConfigFileUsed(), readErr)
			v = getViperInstance(app)
		}
	} else if !os.IsNotExist(statErr) {
		err = fmt.Errorf("unable to read the configuration file %s: %w", v.ConfigFileUsed(), statErr)
	}
	cfg := fromViper(v)
	applyFlags(&cfg, app.Flags)
	return cfg, err
}

// fromViper returns the configuration read by v
func fromViper(v *viper.Viper) Config {
	return Config{
		Interface: v.GetString("interface"),
		Bind:      v.GetString("bind"),
		Port:      v.GetInt("port"),
		KeepAlive: v.GetBool("keepAlive"),
		Path:      v.GetString("path"),
		Secure:    v.GetBool("secure"),
		TlsKey:    v.GetString("tls-key"),
		TlsCert:   v.GetString("tls-cert"),
		FQDN:      v.GetString("fqdn"),
		Output:    v.GetString("output"),
		Reversed:  v.GetBool("reversed"),
		Scheme:    v.GetString("scheme"),
		WithName:  v.GetBool("withName"),
	}
}

// applyFlags overrides the configuration with the flags that are set
func applyFlags(cfg *Config, flags application.Flags) {
	if flags.Interface != "" {
		cfg.Interface = flags.Interface
	}
	if flags.Bind != "" {
		cfg.Bind = flags.Bind
	}
	if flags.Port != 0 {
		cfg.Port = flags.Port
	}
	if flags.KeepAlive {
		cfg.KeepAlive = true
	}
	if flags.Path != "" {
		cfg.Path = flags.Path
	}
	if flags.Secure {
		cfg.Secure = true
	}
	if flags.TlsKey != "" {
		cfg.TlsKey = flags.TlsKey
	}
	if flags.TlsCert != "" {
		cfg.TlsCert = flags.TlsCert
	}
	if flags.FQDN != "" {
		cfg.FQDN = flags.FQDN
	}
	if flags.Output != "" {
		cfg.Output = flags.Output
	}
	if flags.Reversed {
		cfg.Reversed = true
	}
	if flags.Scheme != "" {
		cfg.Scheme = flags.Scheme
	}
	if flags.WithName {
		cfg.WithName = true
	}
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
	readEnv(v, app)
	return v
}

// readEnv makes v read the QRCP_* environment variables
func readEnv(v *viper.Viper, app application.App) {
	v.AutomaticEnv()
	v.SetEnvPrefix(app.Name)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
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
		return err
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
			return fmt.Errorf("%s is a directory", input)
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
			v.Set("tls-cert", util.Expand(promptTlsCertString))
		}
		// TLS key
		promptTlsKey := promptui.Prompt{
			Label:    "Choose TLS certificate key. Empty if not using HTTPS.",
			Default:  cfg.TlsKey,
			Validate: pathIsReadableFile,
		}
		if promptTlsKeyString, err := promptTlsKey.Run(); err == nil {
			v.Set("tls-key", util.Expand(promptTlsKeyString))
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
