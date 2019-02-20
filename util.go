package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/claudiodangelis/qr-filetransfer/config"
)

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
func getAddress(cfg *config.Config) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	var candidateInterface *net.Interface
	for _, iface := range ifaces {
		if iface.Name == cfg.Iface {
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
	var userInput string
	fmt.Scanln(&userInput)
	index, err := strconv.Atoi(userInput)
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
	cfg.Iface = candidateInterface.Name
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

// returns size of the file in human-readable form
func humanReadableSizeOf(pathToFile string) string {
	const (
		B  int64 = 1
		KB       = B << 10 // same as B*1024
		MB       = KB << 10
		GB       = MB << 10
		TB       = GB << 10
		PB       = TB << 10
		EB       = PB << 10
		// do not add sizes biger than Exabyte
		// to avoid overflow of int64 that represents file size in os.FileInfo
	)
	fileInfo, err := os.Stat(pathToFile)
	if err != nil {
		return ""
	}
	fileSize := fileInfo.Size()
	convertSize := func(rawSize, targetSize int64) float64 {
		v := rawSize / targetSize
		r := rawSize % targetSize
		return float64(v) + float64(r)/float64(targetSize)
	}
	switch {
	case fileSize > EB:
		return fmt.Sprintf("%.1f EB", convertSize(fileSize, EB))
	case fileSize > PB:
		return fmt.Sprintf("%.1f PB", convertSize(fileSize, PB))
	case fileSize > TB:
		return fmt.Sprintf("%.1f TB", convertSize(fileSize, TB))
	case fileSize > GB:
		return fmt.Sprintf("%.1f GB", convertSize(fileSize, GB))
	case fileSize > MB:
		return fmt.Sprintf("%.1f MB", convertSize(fileSize, MB))
	case fileSize > KB:
		return fmt.Sprintf("%.1f KB", convertSize(fileSize, KB))
	default:
		return fmt.Sprintf("%d B", fileSize)
	}
}
