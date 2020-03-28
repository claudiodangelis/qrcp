package util

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/jhoonb/archivex"
)

// ZipFiles and return the resulting zip's filename
func ZipFiles(files []string) (string, error) {
	zip := new(archivex.ZipFile)
	tmpfile, err := ioutil.TempFile("", "qrcp")
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	if err := os.Rename(tmpfile.Name(), tmpfile.Name()+".zip"); err != nil {
		return "", err
	}
	zip.Create(tmpfile.Name() + ".zip")
	for _, file := range files {
		f, err := os.Stat(file)
		if err != nil {
			return "", err
		}
		if f.IsDir() {
			zip.AddAll(file, true)
		} else {
			zip.AddFile(file)
		}
	}
	if err := zip.Close(); err != nil {
		return "", nil
	}
	return zip.Name, nil
}

// GetRandomURLPath returns a random string of 4 alphanumeric characters
func GetRandomURLPath() string {
	timeNum := time.Now().UTC().UnixNano()
	alphaString := strconv.FormatInt(timeNum, 36)
	return alphaString[len(alphaString)-4:]
}

// GetSessionID returns a base64 encoded string of 40 random characters
func GetSessionID() (string, error) {
	randbytes := make([]byte, 40)
	if _, err := io.ReadFull(rand.Reader, randbytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(randbytes), nil
}

// GetAddress returns the address of the network interface to
// bind the server to. The first time is run it prompts a
// dialog to choose which network interface should be used
// for the transfer
func GetInterfaceAddress(ifaceString string) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	var candidateInterface *net.Interface
	for _, iface := range ifaces {
		if iface.Name == ifaceString {
			candidateInterface = &iface
			break
		}
	}
	if candidateInterface != nil {
		ip, err := FindIP(*candidateInterface)
		if err != nil {
			return "", err
		}
		return ip, nil
	}

	filteredIfaces := filterInterfaces(ifaces)

	if len(filteredIfaces) == 0 {
		return "", errors.New("no network interface available")
	}
	if len(filteredIfaces) == 1 {
		candidateInterface = &filteredIfaces[0]
		ip, err := FindIP(*candidateInterface)
		if err != nil {
			return "", err
		}
		return ip, nil
	}
	// TODO: This should be managed by a library
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
	ip, err := FindIP(*candidateInterface)
	if err != nil {
		return "", err
	}
	return ip, nil
}

// FindIP returns the IP address of the passed interface, and an error
func FindIP(iface net.Interface) (string, error) {
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

func filterInterfaces(ifaces []net.Interface) []net.Interface {
	filtered := []net.Interface{}
	var re = regexp.MustCompile(`^(veth|br\-|docker|lo|EHC|XHC|bridge|gif|stf|p2p|awdl|utun|tun|tap)`)
	for _, iface := range ifaces {
		if re.MatchString(iface.Name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		filtered = append(filtered, iface)
	}
	return filtered
}

// ReadFilenames from dir
func ReadFilenames(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	// create array of names of files which are stored in dir
	// used later to set valid name for received files
	filenames := make([]string, len(files))
	for _, fi := range files {
		filenames = append(filenames, fi.Name())
	}
	return filenames
}
