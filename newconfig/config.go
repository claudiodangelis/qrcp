package newconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/claudiodangelis/qrcp/application"
	"github.com/spf13/viper"
)

type Config struct {
	Interface string
	Port      int
	KeepAlive bool
	Path      string
	Secure    bool
	TlsKey    string
	TlsCert   string
	FQDN      string
	Output    string
}

// TODO: don't leave this here
const interactive bool = false

func New(app application.App) Config {
	v := viper.New()
	var err error
	cfg := Config{}
	v.SetConfigType("yml")
	if app.Flags.Config != "" {
		v.SetConfigFile(app.Flags.Config)
	} else {
		p := filepath.Join(xdg.ConfigHome, app.Name, fmt.Sprintf("%s.yml", app.Name))
		v.SetConfigFile(p)
	}
	_, err = os.Stat(v.ConfigFileUsed())
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(v.ConfigFileUsed()), os.ModeDir); err != nil {
			panic(err)
		}
		file, err := os.Create(v.ConfigFileUsed())
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}
	v.SetEnvPrefix(strings.ToUpper(app.Name))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	// Load file
	cfg.Interface = v.GetString("interface")
	cfg.Port = v.GetInt("port")
	cfg.KeepAlive = v.GetBool("keepAlive")
	cfg.Path = v.GetString("path")
	cfg.Secure = v.GetBool("secure")
	cfg.TlsKey = v.GetString("tlsKey")
	cfg.TlsCert = v.GetString("tlsCert")
	cfg.FQDN = v.GetString("fqdn")
	cfg.Output = v.GetString("output")

	// Override
	if app.Flags.Interface != "" {
		cfg.Interface = app.Flags.Interface
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

	// Discover interface if it's not been set yet
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

	return cfg
}
