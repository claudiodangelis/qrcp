package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"time"

	"github.com/briandowns/spinner"
	"github.com/claudiodangelis/qrcp/version"
	"github.com/spf13/cobra"
)

type githubRelease struct {
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
}

func getLatest() (*githubRelease, error) {
	var result []githubRelease
	resp, err := http.Get("https://api.github.com/repos/claudiodangelis/qrcp/releases?per_page=1")
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return &result[0], nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version number and build information.",
	Run: func(c *cobra.Command, args []string) {
		fmt.Println("Current version:", version.String())
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " checking the latest available version"
		s.Start()
		latest, err := getLatest()
		s.Stop()
		if err != nil {
			fmt.Println("Unable to get latest available version from Github")
		} else {
			fmt.Printf("Latest available version: %s [date: %s]\n", latest.Name, latest.PublishedAt)
		}
	},
}
