package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudiodangelis/qrcp/application"
)

func appWithConfig(path string) application.App {
	app := application.New()
	app.Flags.Config = path
	return app
}

func TestLoadDoesNotCreateTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qrcp", "config.yml")
	cfg, err := Load(appWithConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interface != "" || cfg.Port != 0 {
		t.Errorf("unexpected configuration %+v", cfg)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("Load created %s", filepath.Dir(path))
	}
}

func TestLoadReadsTheFileAndAppliesFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("interface: wlan0\nport: 8080\nfqdn: example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := appWithConfig(path)
	app.Flags.Port = 9090
	cfg, err := Load(app)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interface != "wlan0" || cfg.Port != 9090 || cfg.FQDN != "example.com" {
		t.Errorf("unexpected configuration %+v", cfg)
	}
	saved, _ := os.ReadFile(path)
	if string(saved) != "interface: wlan0\nport: 8080\nfqdn: example.com\n" {
		t.Errorf("Load changed the file: %q", saved)
	}
}

func TestLoadReturnsErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("interface: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := appWithConfig(path)
	app.Flags.Interface = "eth0"
	cfg, err := Load(app)
	if err == nil {
		t.Error("Load accepted a malformed file")
	}
	if cfg.Interface != "eth0" {
		t.Errorf("the flags were not applied: %+v", cfg)
	}
}

func TestLoadReadsEnvironmentVariables(t *testing.T) {
	t.Setenv("QRCP_PORT", "7070")
	cfg, err := Load(appWithConfig(filepath.Join(t.TempDir(), "config.yml")))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 7070 {
		t.Errorf("port %d, want 7070 from QRCP_PORT", cfg.Port)
	}
}

func TestLoadWithoutExtension(t *testing.T) {
	t.Setenv("QRCP_INTERFACE", "wlan0")
	app := appWithConfig(filepath.Join(t.TempDir(), "noext"))
	app.Flags.Port = 8080
	cfg, err := Load(app)
	if err == nil {
		t.Error("Load accepted a file without extension")
	}
	if cfg.Port != 8080 || cfg.Interface != "wlan0" {
		t.Errorf("the flags or the environment were not applied: %+v", cfg)
	}
}
