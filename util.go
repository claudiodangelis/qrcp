package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// debug prints its argument if the -debug flag is passed
func debug(args ...string) {
	if *debugFlag == true {
		log.Println(args)
	}
}

// findIP returns the IP address of the passed interface, and an error
func findIP(iface net.Interface) (string, error) {
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
			return "[" + ipnet.IP.String() + "]", nil
		}
	}
	return "", errors.New("Unable to find an IP for this interface")
}

// getAddress returns the address of the network interface to
// bind the server to. The first time is run it prompts a
// dialog to choose which network interface should be used
// for the transfer
func getAddress(config *Config) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	var candidateInterface *net.Interface
	for _, iface := range ifaces {
		if iface.Name == config.Iface {
			candidateInterface = &iface
			break
		}
	}
	if candidateInterface != nil {
		ip, err := findIP(*candidateInterface)
		if err != nil {
			return "", err
		}
		return ip, nil
	}
	fmt.Println("Choose the network interface to use (type the number):")
	var filteredIfaces []net.Interface
	for _, iface := range ifaces {
		// TODO: Replace the following with a regexp
		if strings.HasPrefix(iface.Name, "veth") {
			continue
		}
		if strings.HasPrefix(iface.Name, "br-") {
			continue
		}
		if strings.HasPrefix(iface.Name, "docker") {
			continue
		}
		if iface.Name == "lo" {
			continue
		}
		filteredIfaces = append(filteredIfaces, iface)
	}
	for n, iface := range filteredIfaces {
		fmt.Printf("[%d] %s\n", n, iface.Name)
	}
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	index, err := strconv.Atoi(strings.Trim(text, "\n\r"))
	if err != nil {
		return "", err
	}
	if index+1 > len(filteredIfaces) {
		return "", errors.New("Wrong number")
	}
	candidateInterface = &filteredIfaces[index]
	ip, err := findIP(*candidateInterface)
	if err != nil {
		return "", err
	}
	config.Iface = candidateInterface.Name
	return ip, nil
}

// shouldBeZipped returns a boolean value indicating if the
// content should be zipped or not, and an error.
// The content should be zipped if:
// 1. the user passed the `-zip` flag
// 2. there are more than one file
// 3. the file is a directory
func shouldBeZipped(args []string) (bool, error) {
	if *zipFlag == true {
		return true, nil
	}
	if len(args) > 1 {
		return true, nil
	}
	file, err := os.Stat(args[0])
	if err != nil {
		return false, err
	}
	if file.IsDir() {
		return true, nil
	}
	return false, nil
}
