package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/jhoonb/archivex"
	"github.com/mdp/qrterminal"
	"github.com/phayes/freeport"
	"github.com/solarnz/wireless"
)

func getAddress() (string, error) {
	ifaces, err := wireless.NetworkInterfaces()
	if err != nil {
		return "", err
	}
	if len(ifaces) == 0 {
		return "", errors.New("No wireless network found")
	}
	if len(ifaces) > 1 {
		log.Println("Warning: more than one wireless network found.")
		log.Println("Defaulting to the first found")
	}
	iface := ifaces[0]
	i, err := net.InterfaceByName(string(iface))
	if err != nil {
		return "", errors.New(err.Error())
	}
	addrs, err := i.Addrs()
	if err != nil {
		return "", errors.New(err.Error())
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			return ipnet.IP.String(), nil
		}
	}
	return "", errors.New("Unable to find a suitable address")
}

func getPort() string {
	return strconv.Itoa(freeport.GetPort())
}

func getTargetToServe(args []string) (Target, error) {
	toBeZipped := *zipFlag
	var target Target

	if len(flag.Args()) == 1 {
		info, err := os.Stat(flag.Args()[0])
		if err != nil {
			return target, err
		}
		if info.IsDir() {
			toBeZipped = true
		}
	}

	if len(flag.Args()) == 1 && !toBeZipped {
		absolutePath, err := filepath.Abs(flag.Args()[0])
		if err != nil {
			return target, err
		}
		if _, err := os.Stat(absolutePath); err != nil {
			return target, err
		}
		// Copy file
		from, err := os.Open(absolutePath)
		if err != nil {
			return target, err
		}
		defer from.Close()
		tmpfile, err := ioutil.TempFile("", "qr-filetransfer")
		if err != nil {
			return target, err
		}
		defer tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		to, err := os.OpenFile(tmpfile.Name(), os.O_RDWR, 0666)
		if err != nil {
			return target, err
		}
		defer to.Close()
		if _, err := io.Copy(to, from); err != nil {
			return target, err
		}
		target.name = path.Base(flag.Args()[0])
		target.path = tmpfile.Name()
		return target, nil
	}
	// All here should be zipped
	tmpfile, err := ioutil.TempFile("", "qr-filetransfer")
	if err != nil {
		return target, err
	}
	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	zip := new(archivex.ZipFile)
	zip.Create(tmpfile.Name() + ".zip")
	for _, f := range flag.Args() {
		info, err := os.Stat(f)
		if err != nil {
			return target, err
		}
		if info.IsDir() {
			zip.AddAll(f, true)
		} else {
			zip.AddFile(f)
		}
	}
	zip.Close()
	target.name = "Archive.zip"
	if len(flag.Args()) == 1 {
		target.name = path.Base(path.Clean(flag.Args()[0])) + ".zip"
	}
	target.path = tmpfile.Name() + ".zip"
	return target, nil
}

// Target is the file to be transfered
type Target struct {
	name string
	path string
}

// Remove the target from the filesystem
func (t *Target) Remove() error {
	log.Println("About to remove", t.path)
	os.Exit(0)
	return os.Remove(t.path)
}

var zipFlag = flag.Bool("zip", false, "target should be zipped")

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatalln(errors.New(`Example usage:

qr-filetransfer /path/to/file

qr-filetransfer /path/to/file1 /path/to/file2

qr-filetransfer -zip /path/to/bigfile`))
	}
	// Which address should be used
	address, err := getAddress()
	if err != nil {
		log.Fatalln(err)
	}
	// Which port should be used
	port := getPort()
	// What should be transfered
	target, err := getTargetToServe(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}
	qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%s:%s", address, port),
		qrterminal.L, os.Stdout)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename="+target.name)
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		log.Println("I am serving", target.path)
		http.ServeFile(w, r, target.path)
		if err := target.Remove(); err != nil {
			log.Println("Unable to remove target:", err)
		}
		os.Exit(0)
	})
	http.ListenAndServe(fmt.Sprintf("%s:%s", address, port), nil)
}
