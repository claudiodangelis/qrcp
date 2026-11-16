package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInterfaceKind(t *testing.T) {
	// Pretend no interface is reported as wireless by the kernel
	sysClassNet = t.TempDir()
	tests := map[string]Kind{
		"lo":              KindLoopback,
		"lo0":             KindLoopback,
		"eth0":            KindEthernet,
		"enp0s31f6":       KindEthernet,
		"en0":             KindEthernet,
		"wlan0":           KindWiFi,
		"wlp4s0":          KindWiFi,
		"Wi-Fi":           KindWiFi,
		"docker0":         KindVirtual,
		"br-f6721b4c773e": KindVirtual,
		"veth6455020":     KindVirtual,
		"vEthernet (WSL)": KindVirtual,
		"tun0":            KindVPN,
		"utun3":           KindVPN,
		"wg0":             KindVPN,
		"tailscale0":      KindVPN,
		"wwp0s20f0u2":     KindCellular,
		"something":       KindUnknown,
		// Windows
		"Wi-Fi 2":                            KindWiFi,
		"Ethernet 3":                         KindEthernet,
		"Loopback Pseudo-Interface 1":        KindLoopback,
		"Tailscale":                          KindVPN,
		"NordLynx":                           KindVPN,
		"ProtonVPN TUN":                      KindVPN,
		"OpenVPN Wintun":                     KindVPN,
		"ZeroTier One [8056c2e21c]":          KindVPN,
		"VirtualBox Host-Only Network":       KindVirtual,
		"VMware Network Adapter VMnet8":      KindVirtual,
		"Cellular":                           KindCellular,
		"vEthernet (WSL (Hyper-V firewall))": KindVirtual,
		"vEthernet (Default Switch)":         KindVirtual,
		"vEthernet (nat)":                    KindVirtual,
		"vEthernet (DockerNAT)":              KindVirtual,
		"incusbr0":                           KindVirtual,
		"mpqemubr0":                          KindVirtual,
		"waydroid0":                          KindVirtual,
		"vEthernet (External LAN)":           KindEthernet,
	}
	for name, want := range tests {
		if got := InterfaceKind(name); got != want {
			t.Errorf("InterfaceKind(%q) = %s, want %s", name, got, want)
		}
	}
}

func TestInterfaceKindUsesSysfsOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysfs is Linux only")
	}
	sysClassNet = t.TempDir()
	if err := os.MkdirAll(filepath.Join(sysClassNet, "en1", "wireless"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := InterfaceKind("en1"); got != KindWiFi {
		t.Errorf("InterfaceKind(en1) = %s, want %s", got, KindWiFi)
	}
}

func TestBestInterface(t *testing.T) {
	sysClassNet = t.TempDir()
	tests := []struct {
		name   string
		ifaces map[string]string
		want   string
		ok     bool
	}{
		{
			name: "wifi among docker bridges and vpn",
			ifaces: map[string]string{
				"wlp4s0":          "192.168.1.81",
				"br-f6721b4c773e": "172.22.0.1",
				"docker0":         "172.17.0.1",
				"tailscale0":      "100.101.102.103",
				"tun0":            "10.8.0.2",
			},
			want: "wlp4s0", ok: true,
		},
		{
			name:   "wifi and ethernet are ambiguous",
			ifaces: map[string]string{"wlan0": "192.168.1.2", "eth0": "10.0.0.2"},
			ok:     false,
		},
		{
			name:   "public address only",
			ifaces: map[string]string{"eth0": "203.0.113.4"},
			ok:     false,
		},
		{
			name:   "ipv6 only",
			ifaces: map[string]string{"eth0": "[2001:db8::1]"},
			ok:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BestInterface(ListInterfaces(tt.ifaces))
			if ok != tt.ok || (ok && got.Name != tt.want) {
				t.Errorf("BestInterface() = %v, %v; want %s, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestListInterfacesIsSorted(t *testing.T) {
	list := ListInterfaces(map[string]string{"wlan0": "a", "eth0": "b", "lo": "c"})
	if list[0].Name != "eth0" || list[1].Name != "lo" || list[2].Name != "wlan0" {
		t.Errorf("not sorted: %v", list)
	}
}
