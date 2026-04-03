package cmd

import (
	"github.com/claudiodangelis/qrcp/application"
	"github.com/claudiodangelis/qrcp/logger"
	"github.com/claudiodangelis/qrcp/server"
	"github.com/eiannone/keyboard"
	"github.com/spf13/cobra"
)

// listenForQuit starts a goroutine that shuts down the server when "q" or
// Ctrl+C is pressed. The returned function must be deferred by the caller to
// release the keyboard.
func listenForQuit(srv *server.Server, log logger.Logger) func() {
	if err := keyboard.Open(); err != nil {
		log.Print("Warning: keyboard not detected:", err)
		return func() {}
	}
	go func() {
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				break
			}
			if string(char) == "q" || key == keyboard.KeyCtrlC {
				srv.Shutdown()
			}
		}
	}()
	return func() {
		if err := keyboard.Close(); err != nil {
			log.Print("keyboard.Close:", err)
		}
	}
}

var app application.App

func init() {
	app = application.New()
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	configCmd.AddCommand(migrateCmd)
	// Global command flags
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.Quiet, "quiet", "q", false, "only print errors")
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.KeepAlive, "keep-alive", "k", false, "keep server alive after transferring")
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.ListAllInterfaces, "list-all-interfaces", "l", false, "list all available interfaces when choosing the one to use")
	rootCmd.PersistentFlags().IntVarP(&app.Flags.Port, "port", "p", 0, "port to use for the server")
	rootCmd.PersistentFlags().StringVar(&app.Flags.Path, "path", "", "path to use. Defaults to a random string")
	rootCmd.PersistentFlags().StringVarP(&app.Flags.Interface, "interface", "i", "", "network interface to use for the server")
	rootCmd.PersistentFlags().StringVar(&app.Flags.Bind, "bind", "", "address to bind the web server to")
	rootCmd.PersistentFlags().StringVarP(&app.Flags.FQDN, "fqdn", "d", "", "fully-qualified domain name to use for the resulting URLs")
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.Zip, "zip", "z", false, "zip content before transferring")
	rootCmd.PersistentFlags().StringVarP(&app.Flags.Config, "config", "c", "", "path to the config file, defaults to $XDG_CONFIG_HOME/qrcp/config.json")
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.Browser, "browser", "b", false, "display the QR code in a browser window")
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.Secure, "secure", "s", false, "use https connection")
	rootCmd.PersistentFlags().StringVar(&app.Flags.TlsCert, "tls-cert", "", "path to TLS certificate to use with HTTPS")
	rootCmd.PersistentFlags().StringVar(&app.Flags.TlsKey, "tls-key", "", "path to TLS private key to use with HTTPS")
	rootCmd.PersistentFlags().BoolVarP(&app.Flags.Reversed, "reversed", "r", false, "Reverse QR code (black text on white background)")
	// Receive command flags
	receiveCmd.PersistentFlags().StringVarP(&app.Flags.Output, "output", "o", "", "output directory for receiving files")
}

// The root command (`qrcp`) is like a shortcut of the `send` command
var rootCmd = &cobra.Command{
	Use:           "qrcp",
	Args:          cobra.MinimumNArgs(1),
	RunE:          sendCmdFunc,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		rootCmd.PrintErrf("Error: %v\nRun `qrcp help` for help.\n", err)
		return err
	}
	return nil
}
