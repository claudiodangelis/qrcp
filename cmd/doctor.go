package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/doctor"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// errChecksFailed is returned when a check failed. The report already
// explains it, so no other error message is printed.
var errChecksFailed = errors.New("some checks failed")

var statusLabels = map[doctor.Status]string{
	doctor.OK:   "  ok",
	doctor.Info: "info",
	doctor.Warn: "warn",
	doctor.Fail: "FAIL",
}

var statusColors = map[doctor.Status][]color.Attribute{
	doctor.OK:   {color.FgGreen},
	doctor.Info: {color.FgCyan},
	doctor.Warn: {color.FgYellow, color.Bold},
	doctor.Fail: {color.FgRed, color.Bold},
}

// paint returns s with the attributes when colored is true, and s as is
// otherwise
func paint(colored bool, s string, attrs ...color.Attribute) string {
	if !colored {
		return s
	}
	c := color.New(attrs...)
	c.EnableColor()
	return c.Sprint(s)
}

func printReport(w io.Writer, r doctor.Report, colored bool) {
	_, _ = fmt.Fprintln(w, "Network interfaces:")
	for _, i := range r.Interfaces {
		marker := " "
		if i.Name == r.Suggested {
			marker = paint(colored, "*", color.FgGreen, color.Bold)
		}
		_, _ = fmt.Fprintf(w, " %s %-16s %-24s %s\n", marker, i.Name, i.IP, i.Kind)
	}
	if r.Suggested != "" {
		_, _ = fmt.Fprintf(w, "   (%s looks connected to the local network)\n", paint(colored, "*", color.FgGreen, color.Bold))
	}
	_, _ = fmt.Fprintln(w)
	for _, c := range r.Checks {
		label := paint(colored, statusLabels[c.Status], statusColors[c.Status]...)
		_, _ = fmt.Fprintf(w, "[%s] %-9s %s\n", label, c.Name, c.Message)
		if c.Hint != "" {
			_, _ = fmt.Fprintf(w, "       %-9s %s\n", "", paint(colored, "-> "+c.Hint, color.Faint))
		}
	}
	_, _ = fmt.Fprintf(w, "\nMore help: %s\n", doctor.HelpURL)
	if r.Failed() {
		_, _ = fmt.Fprintln(w, "\n"+paint(colored, "Some checks failed: transfers will not work until they are fixed.", color.FgRed, color.Bold))
	}
}

// doctorJSON is the --json flag of doctor
var doctorJSON bool

// jsonReport is the output of doctor --json
type jsonReport struct {
	// Failed is true when a check failed, as the exit code 1
	Failed bool `json:"failed"`
	doctor.Report
	HelpURL string `json:"help_url"`
}

func printJSON(w io.Writer, r doctor.Report) error {
	// Empty lists are printed as [], not null
	if r.Checks == nil {
		r.Checks = []doctor.Check{}
	}
	if r.Interfaces == nil {
		r.Interfaces = []util.Interface{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonReport{Failed: r.Failed(), Report: r, HelpURL: doctor.HelpURL})
}

func doctorCmdFunc(command *cobra.Command, args []string) error {
	// A broken configuration is one of the problems to report, so the
	// network is checked anyway, with the flags only
	cfg, err := config.Load(app)
	report := doctor.Run(cfg, doctor.SystemEnv(app.Flags.ListAllInterfaces))
	if err != nil {
		report.Checks = append([]doctor.Check{{
			Name:    "config",
			Status:  doctor.Fail,
			Message: err.Error(),
			Hint:    "fix the file or the --config path (e.g. one ending in .yml), or remove the file to start over",
		}}, report.Checks...)
	}
	if doctorJSON {
		if err := printJSON(command.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		// color.NoColor is set when stdout is not a terminal, NO_COLOR is
		// set or TERM is dumb; color.Output also translates the colors for
		// old Windows consoles
		printReport(color.Output, report, !color.NoColor)
	}
	if report.Failed() {
		return errChecksFailed
	}
	return nil
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check why the phone can't connect to qrcp",
	Long: `Check this computer's network setup for problems that can stop the phone
from reaching qrcp, and suggest a fix for each one:

  - which network interface qrcp will use, and whether a phone can reach it
  - the address and port the server will listen on, and the host in the URL
  - the firewall, VPNs and WSL networking
  - the TLS certificate and key, with --secure
  - whether the configuration file can be read

doctor only looks at this computer: it can't test the phone or the Wi-Fi
network, so a report without problems doesn't guarantee that the phone
can connect. It never prompts and never changes the configuration.

Pass the flags you use with send or receive to check that setup.

Each check is ok, info (couldn't be fully checked, read it if the phone
still can't connect), warn (likely to cause problems) or fail (transfers
won't work until it's fixed). Exits with code 1 if a check fails.

Scripts and AI agents can use --json, which prints the report as JSON:
{"failed": bool, "checks": [{"name", "status", "message", "hint"}],
"interfaces": [{"name", "ip", "kind"}], "suggested_interface", "help_url"}`,
	Example: `# Check the current configuration
qrcp doctor
# Check a specific interface and port
qrcp doctor --interface wlan0 --port 8080
# Check what --interface any does (it puts the public IP in the URL)
qrcp doctor --interface any
# Machine-readable report, e.g. for scripts and AI agents
qrcp doctor --json`,
	Args: cobra.NoArgs,
	RunE: doctorCmdFunc,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "print the report as JSON")
}
