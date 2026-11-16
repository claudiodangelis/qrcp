package cmd

import (
	"bytes"
	"encoding/json"
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
	printReport(&buf, report, false)
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
	printReport(&buf, report, false)
	if strings.Contains(buf.String(), "Some checks failed") {
		t.Errorf("a passing report says checks failed:\n%s", buf.String())
	}
}

func TestPrintReportColors(t *testing.T) {
	report := doctor.Report{
		Interfaces: []util.Interface{{Name: "wlan0", IP: "192.168.1.2", Kind: util.KindWiFi}},
		Suggested:  "wlan0",
		Checks: []doctor.Check{
			{Name: "interface", Status: doctor.OK, Message: "using wlan0"},
			{Name: "firewall", Status: doctor.Info, Message: "not checked"},
			{Name: "vpn", Status: doctor.Warn, Message: "a VPN is active"},
			{Name: "port", Status: doctor.Fail, Message: "port 80 is in use", Hint: "choose another one"},
		},
	}
	var buf bytes.Buffer
	printReport(&buf, report, false)
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("uncolored report has escape codes:\n%q", buf.String())
	}
	buf.Reset()
	printReport(&buf, report, true)
	out := buf.String()
	for _, want := range []string{
		// The reset sequence depends on the attributes, so only the
		// opening one is checked
		"[\x1b[32m  ok\x1b[", "[\x1b[36minfo\x1b[", "[\x1b[33;1mwarn\x1b[", "[\x1b[31;1mFAIL\x1b[",
		"\x1b[2m-> choose another one\x1b[", " \x1b[32;1m*\x1b[", "\x1b[31;1mSome checks failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%q", want, out)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	report := doctor.Report{
		Interfaces: []util.Interface{{Name: "wlan0", IP: "192.168.1.2", Kind: util.KindWiFi}},
		Suggested:  "wlan0",
		Checks: []doctor.Check{
			{Name: "interface", Status: doctor.OK, Message: "using wlan0"},
			{Name: "port", Status: doctor.Fail, Message: "port 80 is in use", Hint: "choose another one"},
		},
	}
	var buf bytes.Buffer
	if err := printJSON(&buf, report); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Failed     bool             `json:"failed"`
		Checks     []doctor.Check   `json:"checks"`
		Interfaces []util.Interface `json:"interfaces"`
		Suggested  string           `json:"suggested_interface"`
		HelpURL    string           `json:"help_url"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if !got.Failed || len(got.Checks) != 2 || got.Checks[1] != report.Checks[1] ||
		len(got.Interfaces) != 1 || got.Interfaces[0] != report.Interfaces[0] ||
		got.Suggested != "wlan0" || got.HelpURL != doctor.HelpURL {
		t.Errorf("unexpected report: %+v\n%s", got, buf.String())
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("JSON has escape codes:\n%s", buf.String())
	}

	// Empty lists are [], not null, and a passing report isn't failed
	buf.Reset()
	if err := printJSON(&buf, doctor.Report{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"failed": false`, `"checks": []`, `"interfaces": []`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("missing %s in:\n%s", want, buf.String())
		}
	}
}
