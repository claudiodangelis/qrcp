package util

import (
	"errors"
	"net"
	"regexp"
)

// InterfaceNames returns the names of the available interface
func InterfaceNames() ([]string, error) {
	var names []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return names, err
	}
	var re = regexp.MustCompile(`^(veth|br\-|docker|lo|EHC|XHC|bridge|gif|stf|p2p|awdl|utun|tun|tap)`)
	for _, iface := range ifaces {
		if re.MatchString(iface.Name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		names = append(names, iface.Name)
	}
	return names, nil
}

// AddressByInterfaceName returns the address of the passed interface name
func AddressByInterfaceName(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
			// TODO: Explain why this is needed
			return "[" + ipnet.IP.String() + "]", nil
		}
	}
	return "", errors.New("unable to find an IP for this interface")
}
