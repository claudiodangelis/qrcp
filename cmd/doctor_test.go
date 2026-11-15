package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/claudiodangelis/qrcp/doctor"
	"github.com/claudiodangelis/qrcp/util"
)

func TestPrintReport(t *testing.T) {
	report := doctor.Report{
		Interfaces: []util.Interface{
			{Name: "docker0", IP: "172.17.0.1", Kind: util.KindVirtual},
			{Name: "wlan0", IP: "192.168.1.2", Kind: util.KindWiFi},
		},
		Suggested: "wlan0",
		Checks: []doctor.Check{
			{Name: "interface", Status: doctor.OK, Message: "using wlan0"},
			{Name: "port", Status: doctor.Fail, Message: "port 80 is in use", Hint: "choose another one with --port"},
		},
	}
	var buf bytes.Buffer
	printReport(&buf, report)
	out := buf.String()
	for _, want := range []string{
		" * wlan0", "   docker0", "[  ok] interface", "[FAIL] port", "-> choose another one with --port",
		doctor.HelpURL, "Some checks failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	report.Checks = report.Checks[:1]
	buf.Reset()
	printReport(&buf, report)
	if strings.Contains(buf.String(), "Some checks failed") {
		t.Errorf("a passing report says checks failed:\n%s", buf.String())
	}
}
