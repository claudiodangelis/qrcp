package util

import (
	"net"
	"regexp"

	externalip "github.com/glendc/go-external-ip"
)

// Interfaces returns a `name:ip` map of the suitable interfaces found
func Interfaces(listAll bool) (map[string]string, error) {
	names := make(map[string]string)
	ifaces, err := net.Interfaces()
	if err != nil {
		return names, err
	}
	var re = regexp.MustCompile(`^(veth|br\-|docker|lo|EHC|XHC|bridge|gif|stf|p2p|awdl|utun|tun|tap)`)
	for _, iface := range ifaces {
		if !listAll && re.MatchString(iface.Name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		ip, err := FindIP(iface)
		if err != nil {
			continue
		}
		names[iface.Name] = ip
	}
	return names, nil
}

// GetExternalIP of this host
func GetExternalIP() (net.IP, error) {
	consensus := externalip.DefaultConsensus(nil, nil)
	// Get your IP, which is never <nil> when err is <nil>
	ip, err := consensus.ExternalIP()
	if err != nil {
		return nil, err
	}
	return ip, nil
}
