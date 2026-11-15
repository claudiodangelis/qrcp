package doctor

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/fs"
	"math/big"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/claudiodangelis/qrcp/config"
)

var now = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

type fakeListener struct{ net.Listener }

func (fakeListener) Close() error { return nil }

// fakeEnv is a Linux machine with the given interfaces, no ufw, firewalld
// inactive and nothing else installed
func fakeEnv(ifaces map[string]string) Env {
	return Env{
		GOOS:       "linux",
		Interfaces: func() (map[string]string, error) { return ifaces, nil },
		Candidates: func() (map[string]string, error) {
			// Like util.Interfaces(false), which hides lo, docker, tun...
			candidates := map[string]string{}
			for name, ip := range ifaces {
				if !strings.HasPrefix(name, "lo") && !strings.HasPrefix(name, "docker") && !strings.HasPrefix(name, "tun") {
					candidates[name] = ip
				}
			}
			return candidates, nil
		},
		Listen:   func(string) (net.Listener, error) { return fakeListener{}, nil },
		ReadFile: func(string) ([]byte, error) { return nil, fs.ErrNotExist },
		Run:      func(string, ...string) (string, error) { return "inactive\n", errors.New("exit status 3") },
		Now:      func() time.Time { return now },
	}
}

func findCheck(t *testing.T, r Report, name string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("check %s not found in %+v", name, r.Checks)
	return Check{}
}

func has(r Report, name string) bool {
	for _, c := range r.Checks {
		if c.Name == name {
			return true
		}
	}
	return false
}

var home = map[string]string{
	"lo":      "127.0.0.1",
	"wlp4s0":  "192.168.1.81",
	"docker0": "172.17.0.1",
}

func TestHealthyWiFiSetup(t *testing.T) {
	r := Run(config.Config{Interface: "wlp4s0", Port: 8080}, fakeEnv(home))
	if r.Failed() {
		t.Fatalf("unexpected failure: %+v", r.Checks)
	}
	if r.Suggested != "wlp4s0" {
		t.Errorf("suggested %q", r.Suggested)
	}
	if c := findCheck(t, r, "interface"); c.Status != OK {
		t.Errorf("interface: %+v", c)
	}
	if c := findCheck(t, r, "firewall"); c.Status != Info || !strings.Contains(c.Message, "not checked") {
		t.Errorf("firewall: %+v", c)
	}
	if c := findCheck(t, r, "wifi"); c.Status != Info {
		t.Errorf("wifi: %+v", c)
	}
	if has(r, "vpn") || has(r, "wsl") || has(r, "tls") {
		t.Errorf("unexpected checks: %+v", r.Checks)
	}
}

func TestInterfaceProblems(t *testing.T) {
	tests := []struct {
		iface  string
		status Status
		hint   string
	}{
		{"eth9", Fail, "--interface wlp4s0"},
		{"lo", Fail, "--interface wlp4s0"},
		{"docker0", Warn, "--interface wlp4s0"},
	}
	for _, tt := range tests {
		c := findCheck(t, Run(config.Config{Interface: tt.iface}, fakeEnv(home)), "interface")
		if c.Status != tt.status || !strings.Contains(c.Hint, tt.hint) {
			t.Errorf("interface %q: %+v", tt.iface, c)
		}
	}
}

func TestUnconfiguredInterface(t *testing.T) {
	// One candidate: qrcp picks it without asking, and saves it
	c := findCheck(t, Run(config.Config{}, fakeEnv(home)), "interface")
	if c.Status != OK || !strings.Contains(c.Message, "only one") || !strings.Contains(c.Message, "wlp4s0") {
		t.Errorf("one candidate: %+v", c)
	}
	// Several candidates: qrcp asks
	ifaces := map[string]string{"wlp4s0": "192.168.1.81", "virbr0": "192.168.122.1"}
	c = findCheck(t, Run(config.Config{}, fakeEnv(ifaces)), "interface")
	if !strings.Contains(c.Message, "will ask") {
		t.Errorf("several candidates: %+v", c)
	}
	// The only candidate can't be reached
	ifaces = map[string]string{"lo": "127.0.0.1", "virbr0": "192.168.122.1"}
	c = findCheck(t, Run(config.Config{}, fakeEnv(ifaces)), "interface")
	if c.Status != Warn || !strings.Contains(c.Message, "virtual") {
		t.Errorf("unreachable candidate: %+v", c)
	}
	// No candidate
	ifaces = map[string]string{"lo": "127.0.0.1"}
	if c := findCheck(t, Run(config.Config{}, fakeEnv(ifaces)), "interface"); c.Status != Fail {
		t.Errorf("no candidate: %+v", c)
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.Config
		status Status // of the url check; "" when there is none
		text   string
	}{
		{"interface address", config.Config{Interface: "wlp4s0"}, "", ""},
		{"any: public IP", config.Config{Interface: "any"}, Warn, "public IP"},
		{"bind 0.0.0.0: public IP", config.Config{Interface: "wlp4s0", Bind: "0.0.0.0"}, Warn, "public IP"},
		{"any with fqdn", config.Config{Interface: "any", FQDN: "qrcp.example"}, Info, "qrcp.example"},
		{"any with a LAN bind", config.Config{Interface: "any", Bind: "192.168.1.81"}, OK, "192.168.1.81"},
		{"loopback bind", config.Config{Interface: "wlp4s0", Bind: "127.0.0.1"}, Fail, "can't be reached"},
		{"public bind", config.Config{Interface: "wlp4s0", Bind: "203.0.113.5"}, Info, "public address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Run(tt.cfg, fakeEnv(home))
			if tt.status == "" {
				if has(r, "url") {
					t.Errorf("unexpected url check: %+v", r.Checks)
				}
				return
			}
			c := findCheck(t, r, "url")
			if c.Status != tt.status || !strings.Contains(c.Message, tt.text) {
				t.Errorf("got %+v", c)
			}
			if tt.cfg.Interface == "any" {
				if ic := findCheck(t, r, "interface"); ic.Status != Info || strings.Contains(ic.Message, "public") {
					t.Errorf("interface check for any: %+v", ic)
				}
			}
		})
	}
}

func TestNoInterfaces(t *testing.T) {
	r := Run(config.Config{}, fakeEnv(map[string]string{}))
	if !r.Failed() || findCheck(t, r, "interface").Status != Fail {
		t.Errorf("expected failure: %+v", r.Checks)
	}
}

func TestPortInUse(t *testing.T) {
	env := fakeEnv(home)
	env.Listen = func(string) (net.Listener, error) { return nil, errors.New("address already in use") }
	c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Port: 8080}, env), "port")
	if c.Status != Fail || !strings.Contains(c.Hint, "--port") {
		t.Errorf("port: %+v", c)
	}
}

func TestRandomPortSuggestsFixedPort(t *testing.T) {
	c := findCheck(t, Run(config.Config{Interface: "wlp4s0"}, fakeEnv(home)), "port")
	if c.Status != OK || !strings.Contains(c.Hint, "--port") {
		t.Errorf("port: %+v", c)
	}
}

func ufwConf(enabled string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if path == "/etc/ufw/ufw.conf" {
			return []byte("# /etc/ufw/ufw.conf\n#\nENABLED=" + enabled + "\nLOGLEVEL=low\n"), nil
		}
		return nil, fs.ErrNotExist
	}
}

func TestUfw(t *testing.T) {
	env := fakeEnv(home)
	env.ReadFile = ufwConf("yes")
	c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Port: 8080}, env), "firewall")
	if c.Status != Warn || !strings.Contains(c.Message, "ufw") || !strings.Contains(c.Hint, "8080") {
		t.Errorf("enabled: %+v", c)
	}
	// Disabled: the ufw service is still "active", which must not count
	env.ReadFile = ufwConf("no")
	env.Run = func(name string, args ...string) (string, error) {
		if args[len(args)-1] == "ufw" {
			return "active\n", nil
		}
		return "inactive\n", errors.New("exit status 3")
	}
	c = findCheck(t, Run(config.Config{Interface: "wlp4s0"}, env), "firewall")
	if c.Status != Info || !strings.Contains(c.Message, "not active") {
		t.Errorf("disabled: %+v", c)
	}
	// Unreadable setting: unknown
	env.ReadFile = func(string) ([]byte, error) { return nil, fs.ErrPermission }
	c = findCheck(t, Run(config.Config{Interface: "wlp4s0"}, env), "firewall")
	if !strings.Contains(c.Message, "unable to tell") {
		t.Errorf("unreadable: %+v", c)
	}
}

func TestFirewalld(t *testing.T) {
	env := fakeEnv(home)
	env.Run = func(string, ...string) (string, error) { return "active\n", nil }
	if c := findCheck(t, Run(config.Config{Interface: "wlp4s0"}, env), "firewall"); c.Status != Warn || !strings.Contains(c.Message, "firewalld") {
		t.Errorf("firewalld: %+v", c)
	}
}

func TestPortPermissionDenied(t *testing.T) {
	env := fakeEnv(home)
	env.Listen = func(string) (net.Listener, error) {
		return nil, &net.OpError{Op: "listen", Net: "tcp", Err: os.NewSyscallError("bind", syscall.EACCES)}
	}
	c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Port: 80}, env), "port")
	if c.Status != Fail || !strings.Contains(c.Hint, "1024") {
		t.Errorf("port: %+v", c)
	}
}

func TestFirewallDetection(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		run    func(name string, args ...string) (string, error)
		status Status
		text   string
	}{
		{"no systemd", "linux", func(string, ...string) (string, error) {
			return "System has not been booted with systemd\n", errors.New("exit status 1")
		}, Info, "unable to tell"},
		{"macOS on", "darwin", func(string, ...string) (string, error) {
			return "Firewall is enabled. (State = 1)\n", nil
		}, Warn, "macOS firewall"},
		{"macOS blocking all", "darwin", func(string, ...string) (string, error) {
			return "Firewall is blocking all non-essential incoming connections. (State = 2)\n", nil
		}, Warn, "macOS firewall"},
		{"macOS off", "darwin", func(string, ...string) (string, error) {
			return "Firewall is disabled. (State = 0)\n", nil
		}, Info, "is off"},
		{"macOS unexpected output", "darwin", func(string, ...string) (string, error) {
			return "something else\n", nil
		}, Info, "unable to tell"},
		{"Windows on", "windows", func(string, ...string) (string, error) {
			return "3\r\n", nil
		}, Warn, "Windows Defender Firewall"},
		{"Windows off", "windows", func(string, ...string) (string, error) {
			return "0\r\n", nil
		}, Info, "is off"},
		{"Windows without PowerShell", "windows", func(string, ...string) (string, error) {
			return "", errors.New("executable file not found")
		}, Info, "unable to tell"},
		{"other systems", "plan9", nil, Info, "unable to tell"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := fakeEnv(home)
			env.GOOS = tt.goos
			if tt.run != nil {
				env.Run = tt.run
			}
			c := findCheck(t, Run(config.Config{Interface: "wlp4s0"}, env), "firewall")
			if c.Status != tt.status || !strings.Contains(c.Message, tt.text) {
				t.Errorf("got %+v, want %s containing %q", c, tt.status, tt.text)
			}
		})
	}
}

func TestVPN(t *testing.T) {
	ifaces := map[string]string{"wlp4s0": "192.168.1.81", "tun0": "10.8.0.2"}
	if c := findCheck(t, Run(config.Config{Interface: "wlp4s0"}, fakeEnv(ifaces)), "vpn"); c.Status != Warn {
		t.Errorf("vpn: %+v", c)
	}
	// Using the VPN on purpose is reported by the interface check only
	if r := Run(config.Config{Interface: "tun0"}, fakeEnv(ifaces)); has(r, "vpn") {
		t.Errorf("unexpected vpn check: %+v", r.Checks)
	}
}

func procVersion(version string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if path == "/proc/version" {
			return []byte(version), nil
		}
		return nil, fs.ErrNotExist
	}
}

func TestWSL(t *testing.T) {
	env := fakeEnv(map[string]string{"eth0": "172.28.1.2"})
	env.ReadFile = procVersion("Linux version 5.15.153.1-microsoft-standard-WSL2 (root@1234) (gcc) #1 SMP")
	r := Run(config.Config{Interface: "eth0"}, env)
	if c := findCheck(t, r, "wsl"); c.Status != Warn || !strings.Contains(c.Hint, "[wsl2]") || !strings.Contains(c.Hint, "Hyper-V") {
		t.Errorf("wsl: %+v", c)
	}
	// The Linux firewalls don't matter inside WSL
	if c := findCheck(t, r, "firewall"); !strings.Contains(c.Message, "Windows") {
		t.Errorf("firewall in WSL: %+v", c)
	}
	// WSL 1 shares the network of Windows
	env.ReadFile = procVersion("Linux version 4.4.0-19041-Microsoft (Microsoft@Microsoft.com) (gcc version 5.4.0) #1237-Microsoft")
	if r := Run(config.Config{Interface: "eth0"}, env); has(r, "wsl") {
		t.Errorf("WSL 1 reported as WSL 2: %+v", r.Checks)
	}
}

func TestWiFiHintOnMacEthernetNames(t *testing.T) {
	// On macOS, en0 is usually Wi-Fi
	env := fakeEnv(map[string]string{"en0": "192.168.1.20"})
	env.GOOS = "darwin"
	if r := Run(config.Config{Interface: "en0"}, env); !has(r, "wifi") {
		t.Errorf("no client isolation hint: %+v", r.Checks)
	}
}

// certificate returns a PEM certificate and key valid for ips and names
func certificate(t *testing.T, notAfter time.Time, ips []net.IP, names []string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "qrcp test"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     notAfter,
		IPAddresses:  ips,
		DNSNames:     names,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func tlsCheck(t *testing.T, cfg config.Config, certPEM, keyPEM []byte) Check {
	t.Helper()
	env := fakeEnv(home)
	env.ReadFile = func(path string) ([]byte, error) {
		switch path {
		case "cert.pem":
			return certPEM, nil
		case "key.pem":
			return keyPEM, nil
		}
		return nil, errors.New("not found")
	}
	cfg.Interface = "wlp4s0"
	cfg.Secure = true
	return findCheck(t, Run(cfg, env), "tls")
}

func TestTLS(t *testing.T) {
	lan := []net.IP{net.ParseIP("192.168.1.81")}
	year := now.Add(365 * 24 * time.Hour)
	cert, key := certificate(t, year, lan, nil)
	paths := config.Config{TlsCert: "cert.pem", TlsKey: "key.pem"}

	if c := tlsCheck(t, paths, cert, key); c.Status != OK {
		t.Errorf("valid: %+v", c)
	}
	if c := tlsCheck(t, config.Config{}, cert, key); c.Status != Fail {
		t.Errorf("missing paths: %+v", c)
	}
	if c := tlsCheck(t, config.Config{TlsCert: "cert.pem", TlsKey: "nope.pem"}, cert, key); c.Status != Fail {
		t.Errorf("unreadable key: %+v", c)
	}
	_, otherKey := certificate(t, year, lan, nil)
	if c := tlsCheck(t, paths, cert, otherKey); c.Status != Fail {
		t.Errorf("mismatched key: %+v", c)
	}
	expired, expiredKey := certificate(t, now.Add(-time.Minute), lan, nil)
	if c := tlsCheck(t, paths, expired, expiredKey); c.Status != Fail || !strings.Contains(c.Message, "expired") {
		t.Errorf("expired: %+v", c)
	}
	soon, soonKey := certificate(t, now.Add(7*24*time.Hour), lan, nil)
	if c := tlsCheck(t, paths, soon, soonKey); c.Status != Warn {
		t.Errorf("expiring: %+v", c)
	}
	if c := tlsCheck(t, config.Config{TlsCert: "cert.pem", TlsKey: "key.pem", FQDN: "qrcp.example"}, cert, key); c.Status != Warn {
		t.Errorf("wrong host: %+v", c)
	}
	named, namedKey := certificate(t, year, nil, []string{"qrcp.example"})
	if c := tlsCheck(t, config.Config{TlsCert: "cert.pem", TlsKey: "key.pem", FQDN: "qrcp.example"}, named, namedKey); c.Status != OK {
		t.Errorf("fqdn: %+v", c)
	}
	// The host in the URL is the bind address, not the interface's
	if c := tlsCheck(t, config.Config{TlsCert: "cert.pem", TlsKey: "key.pem", Bind: "10.0.0.5"}, cert, key); c.Status != Warn || !strings.Contains(c.Message, "10.0.0.5") {
		t.Errorf("bind: %+v", c)
	}
	// The public IP is not known in advance: never OK
	if c := tlsCheck(t, config.Config{TlsCert: "cert.pem", TlsKey: "key.pem", Bind: "0.0.0.0"}, cert, key); c.Status != Info || !strings.Contains(c.Message, "public IP") {
		t.Errorf("public IP: %+v", c)
	}
}

func TestLoopbackByAddress(t *testing.T) {
	ifaces := map[string]string{"Wi-Fi": "192.168.1.2", "Loopback Pseudo-Interface 1": "127.0.0.1", "weird0": "127.0.0.2"}
	for _, name := range []string{"Loopback Pseudo-Interface 1", "weird0"} {
		if c := findCheck(t, Run(config.Config{Interface: name}, fakeEnv(ifaces)), "interface"); c.Status != Fail {
			t.Errorf("%s: %+v", name, c)
		}
	}
}

func TestWindowsAdapters(t *testing.T) {
	ifaces := map[string]string{"Wi-Fi": "192.168.1.10", "VirtualBox Host-Only Network": "192.168.56.1", "Tailscale": "100.64.0.5"}
	env := fakeEnv(ifaces)
	env.GOOS = "windows"
	r := Run(config.Config{Interface: "Wi-Fi"}, env)
	if r.Suggested != "Wi-Fi" {
		t.Errorf("suggested %q", r.Suggested)
	}
	if c := findCheck(t, r, "vpn"); !strings.Contains(c.Message, "Tailscale") {
		t.Errorf("vpn: %+v", c)
	}
}

// listenRecorder records the addresses doctor tries to listen on
func listenRecorder(env *Env) *[]string {
	var addresses []string
	env.Listen = func(address string) (net.Listener, error) {
		addresses = append(addresses, address)
		return fakeListener{}, nil
	}
	return &addresses
}

func TestPortAddressIsBuiltLikeTheServer(t *testing.T) {
	// server.New listens on fmt.Sprintf("%s:%d", bind, port)
	for bind, want := range map[string]string{"[::1]": "[::1]:0", "::1": "::1:0", "10.0.0.5": "10.0.0.5:0"} {
		env := fakeEnv(home)
		addresses := listenRecorder(&env)
		Run(config.Config{Interface: "wlp4s0", Bind: bind}, env)
		if len(*addresses) != 1 || (*addresses)[0] != want {
			t.Errorf("--bind %s: listened on %v, want %s", bind, *addresses, want)
		}
	}
	// An interface with an IPv6 address only, which util keeps in brackets
	env := fakeEnv(map[string]string{"eth0": "[2001:db8::1]"})
	addresses := listenRecorder(&env)
	Run(config.Config{Interface: "eth0", Port: 8080}, env)
	if (*addresses)[0] != "[2001:db8::1]:8080" {
		t.Errorf("listened on %v", *addresses)
	}
}

func TestPortIsOnlyTriedOnLoopbackOnWindowsAndMacOS(t *testing.T) {
	for _, goos := range []string{"windows", "darwin"} {
		env := fakeEnv(home)
		env.GOOS = goos
		addresses := listenRecorder(&env)
		c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Port: 8080}, env), "port")
		if len(*addresses) != 1 || (*addresses)[0] != "127.0.0.1:8080" || c.Status != Info {
			t.Errorf("%s: listened on %v, check %+v", goos, *addresses, c)
		}
		// A random port needs no check at all
		env = fakeEnv(home)
		env.GOOS = goos
		addresses = listenRecorder(&env)
		if Run(config.Config{Interface: "wlp4s0"}, env); len(*addresses) != 0 {
			t.Errorf("%s: listened on %v for a random port", goos, *addresses)
		}
	}
}

func TestPortErrorHints(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		bind   string
		err    error
		status Status
		hint   string
	}{
		{"bind address not here", "linux", "192.168.1.200", syscall.EADDRNOTAVAIL, Fail, "--bind"},
		{"interface address gone", "linux", "", syscall.EADDRNOTAVAIL, Fail, "run qrcp doctor again"},
		{"in use", "linux", "", syscall.EADDRINUSE, Fail, "another program"},
		{"privileged", "linux", "", syscall.EACCES, Fail, "1024"},
		{"reserved on Windows", "windows", "", syscall.Errno(wsaeAccess), Fail, "excludedportrange"},
		// Only 127.0.0.1 was tried: the real address may be free
		{"in use on loopback on Windows", "windows", "", syscall.Errno(wsaeAddrInUse), Warn, "if qrcp can't start"},
		{"in use on loopback on macOS", "darwin", "", syscall.EADDRINUSE, Warn, "if qrcp can't start"},
	}
	for _, tt := range tests {
		env := fakeEnv(home)
		env.GOOS = tt.goos
		env.Listen = func(string) (net.Listener, error) {
			return nil, &net.OpError{Op: "listen", Net: "tcp", Err: os.NewSyscallError("bind", tt.err)}
		}
		c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: tt.bind, Port: 8080}, env), "port")
		if c.Status != tt.status || !strings.Contains(c.Hint, tt.hint) {
			t.Errorf("%s: %+v", tt.name, c)
		}
	}
}

func TestBindReplacesTheInterfaceAddress(t *testing.T) {
	// The interface is only used to look up an address
	r := Run(config.Config{Interface: "lo", Bind: "192.168.1.50"}, fakeEnv(home))
	if r.Failed() {
		t.Errorf("failed because of the interface: %+v", r.Checks)
	}
	if c := findCheck(t, r, "interface"); !strings.Contains(c.Message, "--bind") {
		t.Errorf("interface: %+v", c)
	}
	// It must still exist, or the server can't start
	if r := Run(config.Config{Interface: "eth9", Bind: "192.168.1.50"}, fakeEnv(home)); !r.Failed() {
		t.Errorf("a missing interface passed: %+v", r.Checks)
	}
	// A name is resolved by the server
	if c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: "myhost.local"}, fakeEnv(home)), "url"); c.Status != Info {
		t.Errorf("hostname: %+v", c)
	}
}

func TestURLWhenTheInterfaceIsUnknown(t *testing.T) {
	// Several candidates and no suggestion: --bind and --fqdn still apply
	ifaces := map[string]string{"eth0": "10.0.0.2", "wlan0": "192.168.1.2"}
	if c := findCheck(t, Run(config.Config{Bind: "0.0.0.0"}, fakeEnv(ifaces)), "url"); c.Status != Warn || !strings.Contains(c.Message, "public IP") {
		t.Errorf("bind 0.0.0.0: %+v", c)
	}
	if c := findCheck(t, Run(config.Config{Interface: "eth9", FQDN: "qrcp.example"}, fakeEnv(ifaces)), "url"); !strings.Contains(c.Message, "qrcp.example") {
		t.Errorf("fqdn: %+v", c)
	}
}

func TestMacFirewallHints(t *testing.T) {
	for state, want := range map[string]string{"1": "allow incoming connections for qrcp", "2": "Block all incoming connections"} {
		env := fakeEnv(home)
		env.GOOS = "darwin"
		env.Run = func(string, ...string) (string, error) { return "Firewall is enabled. (State = " + state + ")\n", nil }
		c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Port: 8080}, env), "firewall")
		if c.Status != Warn || !strings.Contains(c.Hint, want) || strings.Contains(c.Hint, "port 8080") {
			t.Errorf("state %s: %+v", state, c)
		}
	}
}

func TestUnusableBindAddresses(t *testing.T) {
	tests := map[string]string{
		"2001:db8::1": "--bind [2001:db8::1]",
		"::":          "0.0.0.0",
		"[::]":        "0.0.0.0",
		"localhost":   "remove --bind",
	}
	for bind, hint := range tests {
		c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: bind}, fakeEnv(home)), "url")
		if c.Status != Fail || !strings.Contains(c.Hint, hint) {
			t.Errorf("--bind %s: %+v", bind, c)
		}
	}
	// With brackets it's fine
	if c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: "[2001:db8::1]"}, fakeEnv(home)), "url"); c.Status == Fail {
		t.Errorf("--bind [2001:db8::1]: %+v", c)
	}
}

func TestBindAddressNotOnThisComputerWithoutProbing(t *testing.T) {
	for _, goos := range []string{"darwin", "windows"} {
		env := fakeEnv(home)
		env.GOOS = goos
		c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: "10.9.9.9"}, env), "port")
		if c.Status != Warn || !strings.Contains(c.Message, "10.9.9.9") {
			t.Errorf("%s: %+v", goos, c)
		}
		// An address of an interface is fine
		c = findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: "192.168.1.81"}, env), "port")
		if c.Status != OK {
			t.Errorf("%s, local bind: %+v", goos, c)
		}
	}
}

func TestWSLMirroredNetworking(t *testing.T) {
	const version = "Linux version 5.15.153.1-microsoft-standard-WSL2"
	// wslinfo says so
	env := fakeEnv(map[string]string{"eth0": "172.28.1.2"})
	env.ReadFile = procVersion(version)
	env.Run = func(name string, args ...string) (string, error) {
		if name == "wslinfo" {
			return "mirrored\n", nil
		}
		return "", errors.New("not found")
	}
	r := Run(config.Config{Interface: "eth0"}, env)
	if has(r, "wsl") {
		t.Errorf("mirrored (wslinfo): %+v", r.Checks)
	}
	// The Hyper-V firewall still blocks inbound connections by default
	if c := findCheck(t, r, "firewall"); !strings.Contains(c.Message, "Hyper-V") {
		t.Errorf("firewall in mirrored mode: %+v", c)
	}
	// Without wslinfo, mirrored networking isn't available: NAT, whatever
	// the range, since WSL also uses 192.168.x for it
	for _, ip := range []string{"172.28.1.2", "192.168.240.10"} {
		env = fakeEnv(map[string]string{"eth0": ip})
		env.ReadFile = procVersion(version)
		if r := Run(config.Config{Interface: "eth0"}, env); !has(r, "wsl") {
			t.Errorf("NAT with %s not reported: %+v", ip, r.Checks)
		}
	}
}

func TestUnbracketedIPv6PortHint(t *testing.T) {
	env := fakeEnv(home)
	env.Listen = func(string) (net.Listener, error) { return nil, errors.New("too many colons in address") }
	c := findCheck(t, Run(config.Config{Interface: "wlp4s0", Bind: "2001:db8::1", Port: 8080}, env), "port")
	if !strings.Contains(c.Hint, "--bind [2001:db8::1]") {
		t.Errorf("port: %+v", c)
	}
}

func TestBindProblemsWithFQDN(t *testing.T) {
	// The URL uses the name, but the server still listens on --bind
	r := Run(config.Config{Interface: "wlp4s0", Bind: "127.0.0.1", FQDN: "pc.lan"}, fakeEnv(home))
	if c := findCheck(t, r, "url"); c.Status != Fail || strings.Contains(c.Message, "URL uses it") {
		t.Errorf("loopback bind with fqdn: %+v", c)
	}
	r = Run(config.Config{Interface: "wlp4s0", Bind: "192.168.1.81", FQDN: "pc.lan"}, fakeEnv(home))
	if c := findCheck(t, r, "url"); c.Status != Info || !strings.Contains(c.Message, "pc.lan") {
		t.Errorf("fqdn: %+v", c)
	}
}

func TestPortWhenTheInterfaceIsNotChosenYet(t *testing.T) {
	// Several candidates, none suggested: the address isn't known
	ifaces := map[string]string{"eth0": "10.0.0.2", "wlan0": "192.168.1.2"}
	env := fakeEnv(ifaces)
	addresses := listenRecorder(&env)
	c := findCheck(t, Run(config.Config{Port: 8080}, env), "port")
	if c.Status != Info || len(*addresses) != 0 {
		t.Errorf("port: %+v, listened on %v", c, *addresses)
	}
}
