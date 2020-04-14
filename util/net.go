package util

import (
	"net"
	"regexp"
)

// Interfaces returns a `name:ip` map of the suitable interfaces found
func Interfaces() (map[string]string, error) {
	names := make(map[string]string)
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
		ip, err := FindIP(iface)
		if err != nil {
			return names, err
		}
		names[iface.Name] = ip
	}
	return names, nil
}
