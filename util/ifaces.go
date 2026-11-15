package util

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Kind is the likely type of a network interface, guessed from its name
type Kind string

// Interface kinds
const (
	KindWiFi     Kind = "wifi"
	KindEthernet Kind = "ethernet"
	KindCellular Kind = "cellular"
	KindVPN      Kind = "vpn"
	KindVirtual  Kind = "virtual"
	KindLoopback Kind = "loopback"
	KindUnknown  Kind = "unknown"
)

// Interface describes a network interface usable by the server
type Interface struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Kind Kind   `json:"kind"`
}

type kindNames struct {
	kind  Kind
	names []string
}

// kindKeywords match anywhere in descriptive names, like Windows' "Wi-Fi 2"
// or "VirtualBox Host-Only Network". Order matters: the first match wins.
var kindKeywords = []kindNames{
	{KindLoopback, []string{"loopback"}},
	// Hyper-V NAT switches only: an external switch ("vEthernet (<name>)")
	// carries the LAN address of the host
	{KindVirtual, []string{"vethernet (wsl", "vethernet (default switch)", "vethernet (nat)", "vethernet (dockernat)", "virtualbox", "vmware", "virtual"}},
	{KindVPN, []string{"vpn", "tailscale", "nordlynx", "proton", "wireguard", "wintun", "zerotier", "tap-windows"}},
	{KindCellular, []string{"cellular", "mobile broadband"}},
	{KindWiFi, []string{"wi-fi", "wifi", "wireless", "wlan"}},
	{KindEthernet, []string{"ethernet"}},
}

// kindPrefixes match the beginning of Unix names like "eth0". Order
// matters: the first match wins.
var kindPrefixes = []kindNames{
	{KindLoopback, []string{"lo"}},
	{KindVirtual, []string{"docker", "br-", "veth", "virbr", "vmnet", "vboxnet", "bridge", "cni", "flannel", "podman", "lxc", "lxd", "incusbr", "mpqemubr", "waydroid", "anpi"}},
	{KindVPN, []string{"tun", "tap", "utun", "wg", "ppp", "ipsec", "zt", "gpd", "cscotun"}},
	{KindCellular, []string{"wwan", "wwp", "rmnet", "pdp_ip"}},
	{KindWiFi, []string{"wl", "ath"}},
	{KindEthernet, []string{"eth", "en", "em"}},
}

// sysClassNet is where Linux exposes network interfaces
var sysClassNet = "/sys/class/net"

// InterfaceKind guesses the type of a network interface from its name
func InterfaceKind(name string) Kind {
	// On Linux, the kernel knows whether an interface is wireless
	if runtime.GOOS == "linux" {
		if _, err := os.Stat(filepath.Join(sysClassNet, name, "wireless")); err == nil {
			return KindWiFi
		}
	}
	lower := strings.ToLower(name)
	for _, group := range kindKeywords {
		for _, keyword := range group.names {
			if strings.Contains(lower, keyword) {
				return group.kind
			}
		}
	}
	for _, group := range kindPrefixes {
		for _, prefix := range group.names {
			if strings.HasPrefix(lower, prefix) {
				return group.kind
			}
		}
	}
	return KindUnknown
}

// ListInterfaces returns the interfaces in the `name:ip` map, sorted by name
func ListInterfaces(ifaces map[string]string) []Interface {
	list := make([]Interface, 0, len(ifaces))
	for name, ip := range ifaces {
		list = append(list, Interface{Name: name, IP: ip, Kind: InterfaceKind(name)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// isLocalNetwork reports whether the interface probably connects to the
// local network a phone would be on: a Wi-Fi or wired interface with a
// private IPv4 address
func (i Interface) isLocalNetwork() bool {
	if !LANKind(i.Kind) {
		return false
	}
	ip := net.ParseIP(i.IP)
	return ip != nil && ip.To4() != nil && ip.IsPrivate()
}

// LANKind reports whether an interface of this kind may connect to a local
// network, i.e. it is not loopback, virtual, VPN or cellular. Ethernet
// includes macOS en0, which is usually Wi-Fi.
func LANKind(kind Kind) bool {
	switch kind {
	case KindWiFi, KindEthernet, KindUnknown:
		return true
	}
	return false
}

// BestInterface returns the interface that is clearly the one to use, i.e.
// the only one that looks connected to a local network. It returns false
// when there is no such interface, or more than one.
func BestInterface(ifaces []Interface) (Interface, bool) {
	var best Interface
	found := 0
	for _, iface := range ifaces {
		if iface.isLocalNetwork() {
			best = iface
			found++
		}
	}
	return best, found == 1
}
