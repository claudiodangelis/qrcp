package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/doctor"
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
	// color.NoColor is set when stdout is not a terminal, NO_COLOR is set
	// or TERM is dumb; color.Output also translates the colors for old
	// Windows consoles
	printReport(color.Output, report, !color.NoColor)
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
