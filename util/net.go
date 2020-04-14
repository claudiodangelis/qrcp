package util

import (
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
