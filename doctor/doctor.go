// Package doctor diagnoses the most common reasons why a phone cannot reach
// qrcp: wrong interface or URL, port, firewall, VPN, WSL, Wi-Fi client
// isolation and TLS configuration.
package doctor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/util"
)

// HelpURL points to the documentation about networking issues
const HelpURL = "https://github.com/claudiodangelis/qrcp#troubleshooting"

// wslHelpURL documents WSL networking, including mirrored mode and the
// Hyper-V firewall
const wslHelpURL = "https://learn.microsoft.com/windows/wsl/networking"

// Status of a check
type Status string

// Check statuses
const (
	OK   Status = "ok"
	Info Status = "info"
	Warn Status = "warn"
	Fail Status = "fail"
)

// Check is the result of a single diagnostic
type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Report is the result of all diagnostics
type Report struct {
	Checks     []Check          `json:"checks"`
	Interfaces []util.Interface `json:"interfaces"`
	// Suggested is the interface that looks connected to a local network,
	// empty if there is none or more than one
	Suggested string `json:"suggested_interface,omitempty"`
}

// Failed reports whether any check failed
func (r Report) Failed() bool {
	for _, c := range r.Checks {
		if c.Status == Fail {
			return true
		}
	}
	return false
}

// Env is what the checks read from the system, replaceable in tests
type Env struct {
	GOOS string
	// Interfaces returns all the interfaces that are up and have an address
	Interfaces func() (map[string]string, error)
	// Candidates returns the interfaces qrcp offers when none is configured
	Candidates func() (map[string]string, error)
	Listen     func(address string) (net.Listener, error)
	ReadFile   func(path string) ([]byte, error)
	// Run runs a command and returns its standard output
	Run func(name string, args ...string) (string, error)
	Now func() time.Time
}

// SystemEnv returns the environment of the running system. listAll is the
// --list-all-interfaces flag, which changes the interfaces qrcp offers.
func SystemEnv(listAll bool) Env {
	return Env{
		GOOS:       runtime.GOOS,
		Interfaces: func() (map[string]string, error) { return util.Interfaces(true) },
		Candidates: func() (map[string]string, error) { return util.Interfaces(listAll) },
		Listen:     func(address string) (net.Listener, error) { return net.Listen("tcp", address) },
		ReadFile:   os.ReadFile,
		Run: func(name string, args ...string) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Standard output only: e.g. PowerShell writes progress records
			// to standard error the first time a module is used
			out, err := exec.CommandContext(ctx, name, args...).Output()
			return string(out), err
		},
		Now: time.Now,
	}
}

// target is what the server would listen on and put in the URL, worked out
// like server.New does
type target struct {
	// iface is the interface qrcp would use; nil with "any" or when it is
	// unknown
	iface *util.Interface
	// bind is the address the server would listen on, as server.New uses
	// it (IPv6 addresses of interfaces are in brackets); empty when unknown
	bind string
	// missing is set when qrcp would fail to find an interface
	missing bool
	// publicIP is set when the URL would contain the public IP of the
	// network, looked up when the server starts
	publicIP bool
	// urlHost is the host in the URL, empty when publicIP is set or unknown
	urlHost string
}

// Run runs all checks against the configuration
func Run(cfg config.Config, env Env) Report {
	var report Report
	ifaces, err := env.Interfaces()
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name: "interface", Status: Fail,
			Message: fmt.Sprintf("unable to list network interfaces: %v", err),
		})
		return report
	}
	report.Interfaces = util.ListInterfaces(ifaces)
	if best, ok := util.BestInterface(report.Interfaces); ok {
		report.Suggested = best.Name
	}
	t, check := checkInterface(cfg, report.Interfaces, report.Suggested, env)
	t = resolveURL(cfg, t)
	if cfg.Bind != "" && !t.missing && check.Status != OK {
		// The interface is only used to look up an address, which --bind
		// replaces, so it doesn't matter whether a phone can reach it
		check.Status = Info
		check.Hint = ""
		check.Message += fmt.Sprintf("; with --bind, the server listens on %s instead", cfg.Bind)
	}
	report.Checks = append(report.Checks, check)
	if c, ok := checkURL(cfg, t, report.Suggested); ok {
		report.Checks = append(report.Checks, c)
	}
	report.Checks = append(report.Checks, checkPort(cfg, t, report.Interfaces, env))
	inWSL := isWSL2(env)
	wsl := inWSL && !wslMirrored(env)
	// In both networking modes, the firewalls that matter are on Windows
	report.Checks = append(report.Checks, checkFirewall(cfg, env, inWSL))
	if c, ok := checkVPN(report.Interfaces, t.iface); ok {
		report.Checks = append(report.Checks, c)
	}
	if wsl {
		report.Checks = append(report.Checks, Check{
			Name: "wsl", Status: Warn,
			Message: "running inside WSL 2: with the default NAT networking, the WSL address can't be reached from the phone",
			Hint: "on Windows 11 22H2 or later, enable mirrored networking (networkingMode=mirrored under [wsl2] in %UserProfile%\\.wslconfig, then `wsl --shutdown`) and allow inbound connections in the Hyper-V firewall, see " +
				wslHelpURL + "; or run the Windows build of qrcp",
		})
	}
	if t.iface != nil && cfg.Bind == "" && !t.publicIP && util.LANKind(t.iface.Kind) {
		report.Checks = append(report.Checks, Check{
			Name: "wifi", Status: Info,
			Message: "if the phone can't open the URL although the other checks pass, the Wi-Fi network may isolate its clients (common on guest networks and public hotspots)",
			Hint:    "use a network without client isolation, or connect the computer to the phone's hotspot",
		})
	}
	if cfg.Secure {
		report.Checks = append(report.Checks, checkTLS(cfg, t, env))
	}
	return report
}

func find(ifaces []util.Interface, name string) *util.Interface {
	for i := range ifaces {
		if ifaces[i].Name == name {
			return &ifaces[i]
		}
	}
	return nil
}

func describe(i util.Interface) string {
	return fmt.Sprintf("%s (%s, %s)", i.Name, i.IP, i.Kind)
}

func suggestion(suggested string) string {
	if suggested != "" {
		return fmt.Sprintf("use --interface %s, or set it with `qrcp config`", suggested)
	}
	return "choose one with --interface or `qrcp config`"
}

// checkInterface works out the interface qrcp would use and checks it
func checkInterface(cfg config.Config, ifaces []util.Interface, suggested string, env Env) (target, Check) {
	check := Check{Name: "interface"}
	var t target
	if len(ifaces) == 0 {
		check.Status = Fail
		check.Message = "no network interface with an address is up"
		check.Hint = "connect to the same Wi-Fi network as the phone, or turn on the phone's hotspot and connect to it"
		t.missing = true
		return t, check
	}
	switch cfg.Interface {
	case "":
		return chosenInterface(cfg, ifaces, suggested, env)
	case "any":
		t.bind = "0.0.0.0"
		check.Status = Info
		check.Message = "interface \"any\" listens on all interfaces"
		if cfg.Bind != "" {
			check.Message = fmt.Sprintf("interface \"any\", narrowed by --bind to %s", cfg.Bind)
		}
	default:
		t.iface = find(ifaces, cfg.Interface)
		if t.iface == nil {
			check.Status = Fail
			check.Message = fmt.Sprintf("the configured interface %s is not up or has no address", cfg.Interface)
			check.Hint = suggestion(suggested)
			t.missing = true
			return t, check
		}
		t.bind = t.iface.IP
		check = judgeInterface(*t.iface, suggested)
	}
	return t, check
}

// chosenInterface predicts the interface qrcp picks when none is configured:
// the only candidate, without asking, or else the one the user picks
func chosenInterface(cfg config.Config, ifaces []util.Interface, suggested string, env Env) (target, Check) {
	check := Check{Name: "interface"}
	var t target
	candidates, err := env.Candidates()
	if err != nil || len(candidates) == 0 {
		check.Status = Fail
		check.Message = "no interface is configured, and qrcp won't find one to offer"
		check.Hint = suggestion(suggested)
		t.missing = true
		return t, check
	}
	if len(candidates) == 1 {
		for name := range candidates {
			t.iface = find(ifaces, name)
		}
		if t.iface == nil {
			check.Status = Info
			check.Message = "no interface is configured"
			return t, check
		}
		check = judgeInterface(*t.iface, suggested)
		check.Message = "no interface is configured; qrcp will use the only one it finds, and save it to the configuration: " +
			check.Message
	} else {
		if suggested == "" {
			check.Status = Warn
			check.Message = "no interface is configured, and none clearly looks connected to a local network; qrcp will ask which one to use"
			check.Hint = "pick the interface on the same network as the phone"
			return t, check
		}
		t.iface = find(ifaces, suggested)
		check.Status = OK
		check.Message = fmt.Sprintf("no interface is configured; qrcp will ask which one to use, and %s looks connected to the local network", describe(*t.iface))
		check.Hint = fmt.Sprintf("to skip the question, use --interface %s or `qrcp config`", suggested)
	}
	t.bind = t.iface.IP
	return t, check
}

// judgeInterface checks whether a phone can reach an interface
func judgeInterface(iface util.Interface, suggested string) Check {
	check := Check{Name: "interface", Message: "using " + describe(iface)}
	ip := net.ParseIP(strings.Trim(iface.IP, "[]"))
	switch {
	case iface.Kind == util.KindLoopback || (ip != nil && ip.IsLoopback()):
		check.Status = Fail
		check.Message += ": a loopback interface can't be reached from another device"
		check.Hint = suggestion(suggested)
	case iface.Kind == util.KindVirtual:
		check.Status = Warn
		check.Message += ": it looks like a virtual interface (Docker, VM, bridge), which a phone usually can't reach"
		check.Hint = suggestion(suggested)
	case iface.Kind == util.KindVPN:
		check.Status = Warn
		check.Message += ": it looks like a VPN interface; the phone must be connected to the same VPN"
		check.Hint = "if the phone is on the local network instead, " + suggestion(suggested)
	case iface.Kind == util.KindCellular:
		check.Status = Warn
		check.Message += ": it looks like a mobile broadband interface, which usually doesn't accept incoming connections"
		check.Hint = suggestion(suggested)
	case ip != nil && ip.To4() == nil:
		check.Status = Warn
		check.Message += ": it has an IPv6 address only, which many networks and phones can't reach"
		check.Hint = suggestion(suggested)
	case ip != nil && !ip.IsPrivate():
		check.Status = Info
		check.Message += ": it has a public address, the phone must be able to reach it"
	default:
		check.Status = OK
	}
	return check
}

// resolveURL applies --bind and --fqdn like server.New: --bind replaces the
// address of the interface, and the URL then uses the FQDN if set, the
// public IP if the server listens on all interfaces, or the bind address
func resolveURL(cfg config.Config, t target) target {
	if cfg.Bind != "" {
		t.bind = cfg.Bind
	}
	switch {
	case cfg.FQDN != "":
		t.urlHost = cfg.FQDN
	case t.bind == "0.0.0.0":
		t.publicIP = true
	default:
		t.urlHost = t.bind
	}
	return t
}

// checkURL checks the host in the URL, when it is not just the address of
// the interface, which checkInterface already checked
func checkURL(cfg config.Config, t target, suggested string) (Check, bool) {
	check := Check{Name: "url"}
	switch {
	case t.publicIP:
		check.Status = Warn
		check.Message = "the server listens on all interfaces, so the URL uses the public IP of this network, which qrcp looks up with online services; a phone usually can't reach it"
		check.Hint = "on the same network as the phone, " + suggestion(suggested) + "; from elsewhere, forward the port on the router (not possible behind CGNAT), or set --fqdn to a name that reaches this computer"
	case cfg.FQDN != "" && (cfg.Bind == "" || bindCheck(cfg).Status != Fail):
		check.Status = Info
		check.Message = fmt.Sprintf("the URL uses %s: the phone must be able to resolve it to this computer", cfg.FQDN)
	case cfg.Bind != "":
		return bindCheck(cfg), true
	default:
		return Check{}, false
	}
	return check, true
}

// bindCheck checks the address given with --bind, which the server listens
// on, and which the URL uses unless --fqdn is set
func bindCheck(cfg config.Config) Check {
	check := Check{Name: "url", Message: fmt.Sprintf("--bind %s: the server listens on this address", cfg.Bind)}
	if cfg.FQDN == "" {
		check.Message += ", and the URL uses it"
	}
	ip := net.ParseIP(strings.Trim(cfg.Bind, "[]"))
	switch {
	case ip != nil && ip.IsUnspecified():
		// Only 0.0.0.0 makes the server look up the public IP
		check.Status = Fail
		check.Message += "; a phone can't open it"
		check.Hint = "use the address of the interface on the phone's network, or 0.0.0.0 to listen on all interfaces"
	case unbracketedIPv6(cfg.Bind):
		// server.New appends ":port" as is
		check.Status = Fail
		check.Message += "; it's an IPv6 address without brackets, which makes the URL and the address to listen on invalid"
		check.Hint = fmt.Sprintf("write it in brackets: --bind [%s]", cfg.Bind)
	case ip == nil && strings.EqualFold(cfg.Bind, "localhost"), ip != nil && ip.IsLoopback():
		check.Status = Fail
		check.Message += "; it can't be reached from another device"
		check.Hint = "remove --bind, or use the address of the interface on the phone's network"
	case ip == nil:
		// A name: the server listens on what it resolves to
		check.Status = Info
		check.Message += "; it must resolve to an address of this computer that the phone can reach"
	case ip.To4() == nil:
		check.Status = Warn
		check.Message += "; it's an IPv6 address, which many networks and phones can't reach"
	case !ip.IsPrivate():
		check.Status = Info
		check.Message += "; it's a public address, the phone must be able to reach it"
	default:
		check.Status = OK
	}
	return check
}

// Windows socket errors, which syscall.Errno.Is doesn't map
const (
	wsaeAccess       = 10013 // forbidden, e.g. a port reserved by Windows
	wsaeAddrInUse    = 10048
	wsaeAddrNotAvail = 10049
)

func isErrno(err error, unix syscall.Errno, windows uintptr) bool {
	var errno syscall.Errno
	return errors.Is(err, unix) || (errors.As(err, &errno) && uintptr(errno) == windows)
}

// checkPort checks that the server can listen on the configured port. On
// Windows and macOS, listening on a network address can make the firewall
// ask whether to allow qrcp, and dismissing that question on Windows blocks
// it: there, the port is only tried on the loopback address, and the bind
// address is compared with the addresses of the interfaces instead.
func checkPort(cfg config.Config, t target, ifaces []util.Interface, env Env) Check {
	check := Check{Name: "port"}
	randomPort := func() Check {
		check.Status = OK
		check.Message = "a random port is used for each transfer"
		check.Hint = "if a firewall blocks incoming connections, set a fixed port with --port or `qrcp config`, and allow it once"
		return check
	}
	loopbackOnly := env.GOOS != "linux"
	if loopbackOnly && cfg.Bind != "" && !isLocalAddress(cfg.Bind, ifaces) {
		check.Status = Warn
		check.Message = fmt.Sprintf("--bind %s is not among the addresses of the interfaces of this computer, so the server may fail to start", cfg.Bind)
		check.Hint = "use the address of the interface on the phone's network"
		return check
	}
	if loopbackOnly && cfg.Port == 0 {
		return randomPort()
	}
	host := t.bind
	if host == "" && !loopbackOnly {
		// The address depends on the interface the user will choose
		if cfg.Port == 0 {
			return randomPort()
		}
		check.Status = Info
		check.Message = fmt.Sprintf("port %d can't be checked until an interface is chosen", cfg.Port)
		check.Hint = "run qrcp doctor again with --interface"
		return check
	}
	if loopbackOnly {
		host = "127.0.0.1"
	}
	// The address is built like server.New does
	listener, err := env.Listen(fmt.Sprintf("%s:%d", host, cfg.Port))
	if err != nil {
		check.Status = Fail
		check.Message = fmt.Sprintf("unable to listen on %s port %d: %v", host, cfg.Port, err)
		switch {
		case unbracketedIPv6(cfg.Bind):
			check.Hint = fmt.Sprintf("write the address in brackets: --bind [%s]", cfg.Bind)
		case isErrno(err, syscall.EADDRINUSE, wsaeAddrInUse) && loopbackOnly:
			// The server listens on another address, which may be free
			check.Status = Warn
			check.Message = fmt.Sprintf("port %d is in use on 127.0.0.1; qrcp listens on another address, where it may be free", cfg.Port)
			check.Hint = "if qrcp can't start, choose another port with --port"
		case isErrno(err, syscall.EADDRNOTAVAIL, wsaeAddrNotAvail) && cfg.Bind != "":
			check.Hint = fmt.Sprintf("%s is not an address of this computer: check --bind", host)
		case isErrno(err, syscall.EADDRNOTAVAIL, wsaeAddrNotAvail):
			check.Hint = "the address of the interface changed, or is not ready yet: run qrcp doctor again"
		case isErrno(err, syscall.EADDRINUSE, wsaeAddrInUse):
			check.Hint = "another program is using the port; choose another one with --port"
		case isErrno(err, syscall.EACCES, wsaeAccess) && env.GOOS == "windows":
			check.Hint = "Windows may reserve this port (see `netsh int ipv4 show excludedportrange protocol=tcp`), or another program uses it; choose another one with --port"
		case errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EACCES):
			check.Hint = "ports below 1024 need administrator privileges: choose a port of 1024 or above with --port"
		case cfg.Port != 0:
			check.Hint = "choose another port with --port"
		}
		return check
	}
	_ = listener.Close()
	if cfg.Port == 0 {
		return randomPort()
	}
	check.Status = OK
	check.Message = fmt.Sprintf("port %d is available", cfg.Port)
	if loopbackOnly {
		check.Status = Info
		check.Message = fmt.Sprintf("port %d is free on 127.0.0.1; it was not tried on the network address, to avoid a firewall prompt", cfg.Port)
	}
	return check
}

// unbracketedIPv6 reports whether address is an IPv6 address without the
// brackets that server.New needs to append ":port" to it
func unbracketedIPv6(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() == nil
}

// isLocalAddress reports whether address is one of the addresses of the
// interfaces, or one that listens on all of them. A name is assumed to be
// local, since resolving it is up to the server.
func isLocalAddress(address string, ifaces []util.Interface) bool {
	ip := net.ParseIP(strings.Trim(address, "[]"))
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		return true
	}
	for _, i := range ifaces {
		if other := net.ParseIP(strings.Trim(i.IP, "[]")); other != nil && other.Equal(ip) {
			return true
		}
	}
	return false
}

// checkFirewall looks for an active firewall, which is a common reason why
// the phone can't connect
func checkFirewall(cfg config.Config, env Env, wsl bool) Check {
	check := Check{Name: "firewall"}
	allow := "allow incoming TCP connections"
	if cfg.Port != 0 {
		allow += fmt.Sprintf(" on port %d", cfg.Port)
	} else {
		allow += " on a fixed port (set it with --port)"
	}
	if wsl {
		check.Status = Info
		check.Message = "inside WSL, the firewalls that matter are Windows Defender Firewall and the Hyper-V firewall, which blocks inbound connections to WSL by default; they can't be checked from here"
		check.Hint = "in Windows, " + allow + ", see " + wslHelpURL
		return check
	}
	var state firewallState
	switch env.GOOS {
	case "linux":
		state = linuxFirewall(env)
	case "darwin":
		state = macFirewall(env)
	case "windows":
		state = windowsFirewall(env)
	}
	switch {
	case len(state.active) > 0:
		check.Status = Warn
		check.Message = strings.Join(state.active, ", ") + " is active and may block the phone"
		check.Hint = allow
		if state.hint != "" {
			check.Hint = state.hint
		}
	case state.inactive != "":
		// Other firewalls may still be active, so this is never an OK
		check.Status = Info
		check.Message = state.inactive
		check.Hint = "if the phone can't connect, " + allow
	default:
		check.Status = Info
		check.Message = "unable to tell whether a firewall is active"
		check.Hint = "if the phone can't connect, " + allow
	}
	return check
}

// firewallState is what could be found out about the firewall: the active
// ones, or else a description of what was found inactive. Both are empty
// when nothing could be found out.
type firewallState struct {
	active   []string
	inactive string
	// hint replaces the advice to open the port, for firewalls that don't
	// work by port
	hint string
}

// linuxFirewall checks the two common firewall frontends. Rules added with
// iptables or nftables directly are not detected.
func linuxFirewall(env Env) firewallState {
	var state firewallState
	known := 0
	// The ufw service is "active" even when ufw is disabled, so read the
	// setting instead, which doesn't need root
	if conf, err := env.ReadFile("/etc/ufw/ufw.conf"); err == nil {
		switch ufwEnabled(string(conf)) {
		case "yes":
			state.active = append(state.active, "ufw")
			known++
		case "no":
			known++
		}
	} else if errors.Is(err, os.ErrNotExist) {
		known++ // not installed
	}
	out, _ := env.Run("systemctl", "is-active", "firewalld")
	switch strings.TrimSpace(out) {
	case "active":
		state.active = append(state.active, "firewalld")
		known++
	case "inactive", "failed", "unknown":
		known++
	}
	if len(state.active) == 0 && known == 2 {
		state.inactive = "ufw and firewalld are not active; other firewall rules (iptables, nftables) were not checked"
	}
	return state
}

// ufwEnabled returns the value of ENABLED in ufw.conf: "yes", "no" or ""
func ufwEnabled(conf string) string {
	for _, line := range strings.Split(conf, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && key == "ENABLED" {
			return strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
		}
	}
	return ""
}

// macFirewall reads the state of the macOS application firewall: 0 is off,
// 1 is on, 2 blocks all incoming connections
func macFirewall(env Env) firewallState {
	out, err := env.Run("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate")
	if err != nil {
		return firewallState{}
	}
	// The macOS firewall allows or blocks applications, not ports
	switch {
	case strings.Contains(out, "State = 1"):
		return firewallState{
			active: []string{"the macOS firewall"},
			hint:   "allow incoming connections for qrcp in System Settings > Network > Firewall > Options",
		}
	case strings.Contains(out, "State = 2"):
		return firewallState{
			active: []string{"the macOS firewall, blocking all incoming connections,"},
			hint:   "turn off \"Block all incoming connections\" in System Settings > Network > Firewall > Options, and allow incoming connections for qrcp",
		}
	case strings.Contains(out, "State = 0"):
		return firewallState{inactive: "the macOS firewall is off"}
	}
	return firewallState{}
}

// windowsFirewall counts the enabled Windows Defender Firewall profiles.
// PowerShell is used because, unlike netsh, its output is not translated.
func windowsFirewall(env Env) firewallState {
	out, err := env.Run("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"$ProgressPreference = 'SilentlyContinue'; (Get-NetFirewallProfile | Where-Object { $_.Enabled }).Count")
	if err != nil {
		return firewallState{}
	}
	enabled, err := strconv.Atoi(strings.TrimSpace(out))
	switch {
	case err != nil:
		return firewallState{}
	case enabled > 0:
		return firewallState{active: []string{"Windows Defender Firewall"}}
	default:
		return firewallState{inactive: "Windows Defender Firewall is off"}
	}
}

// checkVPN warns about active VPN interfaces, unless the VPN is the one in use
func checkVPN(ifaces []util.Interface, selected *util.Interface) (Check, bool) {
	var names []string
	for _, i := range ifaces {
		if i.Kind == util.KindVPN && (selected == nil || selected.Name != i.Name) {
			names = append(names, i.Name)
		}
	}
	if len(names) == 0 {
		return Check{}, false
	}
	return Check{
		Name: "vpn", Status: Warn,
		Message: "a VPN looks active (" + strings.Join(names, ", ") + "); some VPNs block connections from the local network",
		Hint:    "if the phone can't connect, allow local network access in the VPN settings or disconnect it",
	}, true
}

// wslMirrored reports whether WSL 2 uses mirrored networking, where WSL has
// the addresses of Windows. It asks wslinfo; versions of WSL without it don't
// support mirrored networking, so its absence means NAT.
func wslMirrored(env Env) bool {
	out, err := env.Run("wslinfo", "--networking-mode")
	return err == nil && strings.TrimSpace(out) == "mirrored"
}

// isWSL2 reports whether qrcp runs inside WSL 2. WSL 1 shares the network
// of Windows, so it doesn't have the NAT problem.
func isWSL2(env Env) bool {
	if env.GOOS != "linux" {
		return false
	}
	version, err := env.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	v := strings.ToLower(string(version))
	return strings.Contains(v, "microsoft-standard") || strings.Contains(v, "wsl2")
}

// checkTLS validates the certificate and key used with --secure
func checkTLS(cfg config.Config, t target, env Env) Check {
	check := Check{Name: "tls"}
	hint := "set them with --tls-cert and --tls-key or `qrcp config`"
	if cfg.TlsCert == "" || cfg.TlsKey == "" {
		check.Status = Fail
		check.Message = "HTTPS is enabled but the TLS certificate or key path is not set"
		check.Hint = hint
		return check
	}
	certPEM, err := env.ReadFile(cfg.TlsCert)
	if err != nil {
		check.Status = Fail
		check.Message = fmt.Sprintf("unable to read the TLS certificate: %v", err)
		check.Hint = hint
		return check
	}
	keyPEM, err := env.ReadFile(cfg.TlsKey)
	if err != nil {
		check.Status = Fail
		check.Message = fmt.Sprintf("unable to read the TLS key: %v", err)
		check.Hint = hint
		return check
	}
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		check.Status = Fail
		check.Message = fmt.Sprintf("invalid TLS certificate or key: %v", err)
		check.Hint = "check that the certificate and the key are PEM files and belong together"
		return check
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		check.Status = Fail
		check.Message = fmt.Sprintf("invalid TLS certificate: %v", err)
		return check
	}
	now := env.Now()
	switch {
	case now.After(cert.NotAfter):
		check.Status = Fail
		check.Message = fmt.Sprintf("the TLS certificate expired on %s", cert.NotAfter.Format("2006-01-02"))
		check.Hint = "create a new certificate"
		return check
	case now.Before(cert.NotBefore):
		check.Status = Fail
		check.Message = fmt.Sprintf("the TLS certificate is not valid before %s", cert.NotBefore.Format("2006-01-02"))
		check.Hint = "check the clock of this computer"
		return check
	}
	expiry := cert.NotAfter.Format("2006-01-02")
	if t.urlHost == "" {
		// The host in the URL is not known in advance
		check.Status = Info
		check.Message = fmt.Sprintf("the TLS certificate is valid until %s, but whether it covers the host in the URL can't be checked", expiry)
		if t.publicIP {
			check.Message = fmt.Sprintf("the TLS certificate is valid until %s, but the URL uses the public IP, looked up when qrcp starts, so whether the certificate covers it can't be checked", expiry)
		}
		return check
	}
	if err := cert.VerifyHostname(strings.Trim(t.urlHost, "[]")); err != nil {
		check.Status = Warn
		check.Message = fmt.Sprintf("the TLS certificate is not valid for %s, which is in the URL; the phone will show a warning", t.urlHost)
		check.Hint = "create a certificate that includes it, e.g. with mkcert"
		return check
	}
	check.Status = OK
	check.Message = fmt.Sprintf("the TLS certificate is valid until %s and covers %s", expiry, t.urlHost)
	if cert.NotAfter.Sub(now) < 30*24*time.Hour {
		check.Status = Warn
		check.Message = fmt.Sprintf("the TLS certificate expires soon, on %s", expiry)
		check.Hint = "create a new certificate"
	}
	return check
}
