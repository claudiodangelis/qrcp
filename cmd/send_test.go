package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSendFlags(t *testing.T) {
	// The root command is a shortcut of send, so both take the send flags
	for _, c := range []*cobra.Command{rootCmd, sendCmd} {
		scheme := c.Flags().Lookup("scheme")
		if scheme == nil || scheme.Value.Type() != "string" || scheme.DefValue != "" {
			t.Errorf("%s: unexpected --scheme flag %+v", c.Name(), scheme)
		}
		withName := c.Flags().Lookup("with-name")
		if withName == nil || withName.Value.Type() != "bool" || withName.DefValue != "false" {
			t.Errorf("%s: unexpected --with-name flag %+v", c.Name(), withName)
		}
		if short := c.Flags().ShorthandLookup("w"); short == nil || short.Name != "with-name" {
			t.Errorf("%s: -w is not --with-name", c.Name())
		}
	}
	// Receiving has no file to name and no URL for other apps
	for _, name := range []string{"scheme", "with-name"} {
		if receiveCmd.Flags().Lookup(name) != nil {
			t.Errorf("receive takes --%s", name)
		}
	}
}

func TestSendFlagsAreParsed(t *testing.T) {
	saved := app.Flags
	t.Cleanup(func() { app.Flags = saved })
	for _, c := range []*cobra.Command{rootCmd, sendCmd} {
		app.Flags.Scheme, app.Flags.WithName = "", false
		if err := c.ParseFlags([]string{"--scheme", "zpkd1", "-w"}); err != nil {
			t.Fatalf("%s: %v", c.Name(), err)
		}
		if app.Flags.Scheme != "zpkd1" || !app.Flags.WithName {
			t.Errorf("%s: parsed %+v", c.Name(), app.Flags)
		}
	}
}
