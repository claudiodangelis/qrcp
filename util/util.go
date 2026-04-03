package util

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jhoonb/archivex"
)

// Expand tilde in paths
func Expand(input string) string {
	if runtime.GOOS == "windows" {
		return input
	}
	usr, err := user.Current()
	if err != nil {
		return input
	}
	dir := usr.HomeDir
	if input == "~" {
		input = dir
	} else if strings.HasPrefix(input, "~/") {
		input = filepath.Join(dir, input[2:])
	}
	return input
}

// ZipFiles and return the resulting zip's filename
func ZipFiles(files []string) (string, error) {
	zip := new(archivex.ZipFile)
	tmpfile, err := os.CreateTemp("", "qrcp*.zip")
	if err != nil {
		return "", err
	}
	if err := zip.CreateWriter(tmpfile.Name(), tmpfile); err != nil {
		return "", err
	}

	for _, filename := range files {
		fileinfo, err := os.Stat(filename)
		if err != nil {
			return "", err
		}
		if fileinfo.IsDir() {
			if err := zip.AddAll(filename, true); err != nil {
				return "", err
			}
		} else {
			file, err := os.Open(filename)
			if err != nil {
				return "", err
			}
			err = zip.Add(filename, file, fileinfo)
			_ = file.Close()
			if err != nil {
				return "", err
			}
		}
	}

	if err := zip.Writer.Close(); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return zip.Name, nil
}

// GetRandomURLPath returns a random string of 4 URL-safe characters
func GetRandomURLPath() string {
	b := make([]byte, 3) // 3 bytes encodes to exactly 4 base64 chars
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// GetSessionID returns a base64 encoded string of 40 random bytes
func GetSessionID() (string, error) {
	b := make([]byte, 40)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// GetInterfaceAddress returns the address of the network interface to
// bind the server to. If the interface is "any", it will return 0.0.0.0.
// If no interface is found with that name, an error is returned
func GetInterfaceAddress(ifaceString string) (string, error) {
	if ifaceString == "any" {
		return "0.0.0.0", nil
	}
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
	return "", errors.New("unable to find interface")
}

// FindIP returns the IP address of the passed interface, and an error
func FindIP(iface net.Interface) (string, error) {
	var ip string
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
				ip = ipnet.IP.String()
				continue
			}
			// Use IPv6 only if an IPv4 hasn't been found yet.
			// This is eventually overwritten with an IPv4, if found (see above)
			if ip == "" {
				ip = "[" + ipnet.IP.String() + "]"
			}
		}
	}
	if ip == "" {
		return "", errors.New("unable to find an IP for this interface")
	}
	return ip, nil
}

// ReadFilenames from dir
func ReadFilenames(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	filenames := make([]string, 0, len(files))
	for _, fi := range files {
		filenames = append(filenames, fi.Name())
	}
	return filenames, nil
}
