package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// debug prints its argument if the -debug flag is passed
// and -quiet flag is not passed
func debug(args ...string) {
	if *quietFlag == false && *debugFlag == true {
		log.Println(args)
	}
}

// info prints its argument if the -quiet flag is not passed
func info(args ...interface{}) {
	if *quietFlag == false {
		fmt.Println(args...)
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

	var filteredIfaces []net.Interface
	var re = regexp.MustCompile(`^(veth|br\-|docker|lo|EHC|XHC|bridge|gif|stf|p2p|awdl|utun|tun|tap)`)
	for _, iface := range ifaces {
		if re.MatchString(iface.Name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		filteredIfaces = append(filteredIfaces, iface)
	}
	if len(filteredIfaces) == 0 {
		return "", errors.New("no network interface available")
	}
	if len(filteredIfaces) == 1 {
		candidateInterface = &filteredIfaces[0]
		ip, err := findIP(*candidateInterface)
		if err != nil {
			return "", err
		}
		return ip, nil
	}
	fmt.Println("Choose the network interface to use (type the number):")
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

// getRandomURLPath returns a random string of 4 alphanumeric characters
func getRandomURLPath() string {
	timeNum := time.Now().UTC().UnixNano()
	alphaString := strconv.FormatInt(timeNum, 36)
	return alphaString[len(alphaString)-4:]
}

// getSessionID returns a base64 encoded string of 40 random characters
func getSessionID() (string, error) {
	randbytes := make([]byte, 40)
	if _, err := io.ReadFull(rand.Reader, randbytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(randbytes), nil
}
