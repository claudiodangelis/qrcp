package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mdp/qrterminal"
	"github.com/phayes/freeport"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var uploadFlag = flag.Bool("upload", false, "upload a file rather than download")
var debugFlag = flag.Bool("debug", false, "increase verbosity")

func main() {
	flag.Parse()
	config := LoadConfig()
	if *forceFlag == true {
		config.Delete()
		config = LoadConfig()
	}

	// Check how many arguments are passed
	if len(flag.Args()) == 0 {
		log.Fatalln("At least one argument is required")
	}

	// Get addresses
	address, err := getAddress(&config)
	if err != nil {
		log.Fatalln(err)
	}

	if err := config.Update(); err != nil {
		log.Println("Unable to update configuration", err)
	}

	// Get a random available port
	port := freeport.GetPort()

	// Check how many arguments are passed
	if *uploadFlag == true {
		path := flag.Args()[0]

		http.HandleFunc("/", uploadPage)

		http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
			upload, _, err := r.FormFile("file")
			if err != nil {
				log.Fatalln("File upload error", err)
			}

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
			if err != nil {
				log.Fatalln(err)
			}

			io.Copy(f, upload)

			upload.Close()
			f.Close()

			os.Exit(0)
		})

		fmt.Println("Scan the following QR to start the upload")
	} else {
		content, err := getContent(flag.Args())
		if err != nil {
			log.Fatalln(err)
		}

		// Define a default handler for the requests
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Disposition",
				"attachment; filename="+content.Name())

			w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
			http.ServeFile(w, r, content.Path)
			if content.ShouldBeDeleted {
				if err := content.Delete(); err != nil {
					log.Println("Unable to delete the content from disk", err)
				}
			}
			os.Exit(0)
		})

		fmt.Println("Scan the following QR to start the download.")
	}

	// Generate the QR code
	fmt.Println("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%s:%d", address, port),
		qrterminal.L, os.Stdout)

	// Start a new server bound to the chosen address on a random port
	log.Fatalln(http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil))
}

func uploadPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprint(w, `<!doctype html>
<html>
<body>
  <div class="box">
    <form method="post" action="/upload" id="form" enctype="multipart/form-data" class="box">
    <label for="file" class="box">
      Upload a File
      <input type="file" id="file" name="file">
    </label>
    </form>
  </div>
  <style>
  #file {
    position: absolute;
    opacity: 0;
    width: 0.1px;
    height: 0.1px;
  }

  html, body {
    height: 100%;
    width: 100%;
  }

  body {
    margin: 0;
  }

  .box {
  display: -webkit-flexbox;
    display: -ms-flexbox;
    display: -webkit-flex;
    display: flex;
    -webkit-flex-align: center;
    -ms-flex-align: center;
    -webkit-align-items: center;
    align-items: center;
    -webkit-justify-content: center;
    justify-content: center;
    text-align: center;
    height: 100vh;
    width: 100%;
  }

  label {
    font: bold 5vh Helvetica, sans-serif;
    margin: auto;
  }
  </style>
  <script type="text/javascript">
  document.getElementById('file').onchange = function() {
    document.getElementById('form').submit();
  };
  </script>
</body>
</html>`)
}
