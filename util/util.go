package util

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Expand tilde in paths
func Expand(input string) string {
	if runtime.GOOS == "windows" {
		return input
	}
	usr, _ := user.Current()
	dir := usr.HomeDir
	if input == "~" {
		input = dir
	} else if strings.HasPrefix(input, "~/") {
		input = filepath.Join(dir, input[2:])
	}
	return input
}

// add folder to the zip file
func addFolderToZip(zipWriter *zip.Writer, source, target string) error {

	//explore the folder and add all to the zip
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = path

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// add files to the zip file
func addFileToZip(zipWriter *zip.Writer, fileToAdd string) error {
	f1, err := os.Open(fileToAdd)
	if err != nil {
		return err
	}
	defer f1.Close()

	w1, err := zipWriter.Create(filepath.Base(fileToAdd))
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	return nil
}

// ZipFiles and return the resulting zip's filename
func ZipFiles(files []string) (string, error) {
	//create temporary file
	tmpfile, err := os.CreateTemp("", "qrcp")
	if err != nil {
		return "", err
	}
	tempFileName := tmpfile.Name() + ".zip"
	tmpfile.Close()
	if err := os.Rename(tmpfile.Name(), tempFileName); err != nil {
		return "", err
	}

	//create zip file
	zipFile, err := os.Create(tempFileName)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	//add files and folder in the zip
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	for _, filename := range files {
		fileinfo, err := os.Stat(filename)
		if err != nil {
			return "", err
		}
		if fileinfo.IsDir() {
			if err := addFolderToZip(zipWriter, filename, tempFileName); err != nil {
				return "", err
			}
		} else {
			if err := addFileToZip(zipWriter, filename); err != nil {
				return "", err
			}
		}
	}
	return tempFileName, nil
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
func ReadFilenames(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	// Create array of names of files which are stored in dir
	// used later to set valid name for received files
	filenames := make([]string, len(files))
	for _, fi := range files {
		filenames = append(filenames, fi.Name())
	}
	return filenames
}
