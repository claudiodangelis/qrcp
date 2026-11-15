package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/doctor"
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

func printReport(w io.Writer, r doctor.Report) {
	_, _ = fmt.Fprintln(w, "Network interfaces:")
	for _, i := range r.Interfaces {
		marker := " "
		if i.Name == r.Suggested {
			marker = "*"
		}
		_, _ = fmt.Fprintf(w, " %s %-16s %-24s %s\n", marker, i.Name, i.IP, i.Kind)
	}
	if r.Suggested != "" {
		_, _ = fmt.Fprintln(w, "   (* looks connected to the local network)")
	}
	_, _ = fmt.Fprintln(w)
	for _, c := range r.Checks {
		_, _ = fmt.Fprintf(w, "[%s] %-9s %s\n", statusLabels[c.Status], c.Name, c.Message)
		if c.Hint != "" {
			_, _ = fmt.Fprintf(w, "       %-9s -> %s\n", "", c.Hint)
		}
	}
	_, _ = fmt.Fprintf(w, "\nMore help: %s\n", doctor.HelpURL)
	if r.Failed() {
		_, _ = fmt.Fprintln(w, "\nSome checks failed: transfers will not work until they are fixed.")
	}
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
	printReport(os.Stdout, report)
	if report.Failed() {
		return errChecksFailed
	}
	return nil
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose network problems",
	Long: "Check the network interfaces, the port, the firewall, VPNs, WSL and the TLS configuration, " +
		"and suggest how to fix what could prevent the phone from reaching qrcp. " +
		"It uses the same configuration and flags as the other commands, never changes the configuration, " +
		"and exits with code 1 if a check fails.",
	Example: `# Check the current configuration
qrcp doctor
# Check a specific interface and port
qrcp doctor --interface wlan0 --port 8080`,
	Args: cobra.NoArgs,
	RunE: doctorCmdFunc,
}
