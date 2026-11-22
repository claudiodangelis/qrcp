package server

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudiodangelis/qrcp/body"
	"github.com/claudiodangelis/qrcp/config"
)

// newConfiguredServer is newTestServer with the given scheme and with-name
func newConfiguredServer(t *testing.T, scheme string, withName bool) *Server {
	t.Helper()
	captureLog(t)
	cfg := config.Config{Interface: "any", Bind: "127.0.0.1", Scheme: scheme, WithName: withName}
	srv, err := New(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Shutdown)
	return srv
}

// sendFile makes srv send a file with the given name and content
func sendFile(t *testing.T, srv *Server, name string, content []byte) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatal(err)
	}
	srv.Send(body.Body{Filename: name, Path: file})
}

// httpURL replaces the scheme of u with the one the server uses
func httpURL(t *testing.T, srv *Server, u string) string {
	t.Helper()
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Scheme = strings.SplitN(srv.BaseURL, "://", 2)[0]
	return parsed.String()
}

func status(t *testing.T, u string) int {
	t.Helper()
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestSendScheme(t *testing.T) {
	tests := []struct {
		custom, protocol, want string
	}{
		{"", "http", "http"},
		{"", "https", "https"},
		{"zpkd1", "http", "zpkd1"},
		{"zpkd1", "https", "zpkd1"},
		{"zpkd1://", "http", "zpkd1"},
		{"a+b-c.d", "http", "a+b-c.d"},
		{"ZPKD1", "http", "ZPKD1"},
	}
	for _, tt := range tests {
		got, err := sendScheme(tt.custom, tt.protocol)
		if err != nil || got != tt.want {
			t.Errorf("sendScheme(%q, %q) = %q, %v, want %q", tt.custom, tt.protocol, got, err, tt.want)
		}
	}
}

func TestSendSchemeRejectsInvalidSchemes(t *testing.T) {
	for _, custom := range []string{"://", "1zpk", "+zpk", "zp kd1", "zpk/d1", "zpk:d1", "zpkd1:", "zpkd1:/", "zpkè", "zpk?", "zpk#"} {
		if got, err := sendScheme(custom, "http"); err == nil {
			t.Errorf("sendScheme(%q) = %q, want an error", custom, got)
		}
	}
}

func TestNewRejectsAnInvalidScheme(t *testing.T) {
	cfg := config.Config{Interface: "any", Bind: "127.0.0.1", Scheme: "not a scheme"}
	if _, err := New(&cfg); err == nil {
		t.Fatal("New accepted an invalid scheme")
	}
}

func TestSchemeOnlyChangesSendURL(t *testing.T) {
	srv := newConfiguredServer(t, "zpkd1", false)
	sendFile(t, srv, "watchface.zpk", []byte("watchface"))
	if !strings.HasPrefix(srv.SendURL, "zpkd1://127.0.0.1:") {
		t.Errorf("SendURL %q does not use the scheme", srv.SendURL)
	}
	if !strings.HasSuffix(srv.SendURL, "/send/"+srv.path) {
		t.Errorf("SendURL %q does not end with the send path", srv.SendURL)
	}
	if !strings.HasPrefix(srv.BaseURL, "http://") {
		t.Errorf("BaseURL %q, used to open the QR code in the browser, does not use http", srv.BaseURL)
	}
	if !strings.HasPrefix(srv.ReceiveURL, "http://") {
		t.Errorf("ReceiveURL %q does not use http", srv.ReceiveURL)
	}
	// The server still speaks http
	resp := get(t, httpURL(t, srv, srv.SendURL))
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(got) != "watchface" {
		t.Errorf("received %q", got)
	}
}

func TestDefaultSendURLHasNoName(t *testing.T) {
	srv := newConfiguredServer(t, "", false)
	sendFile(t, srv, "file.bin", []byte("content"))
	if want := srv.BaseURL + "/send/" + srv.path; srv.SendURL != want {
		t.Errorf("SendURL %q, want %q", srv.SendURL, want)
	}
	// Without --with-name, the URL with the name is not served
	if code := status(t, srv.SendURL+"/file.bin"); code != http.StatusNotFound {
		t.Errorf("URL with the name: status %d, want 404", code)
	}
}

func TestWithNameServesTheFile(t *testing.T) {
	content := bytes.Repeat([]byte("zpk"), 100000)
	srv := newConfiguredServer(t, "", true)
	sendFile(t, srv, "watchface.zpk", content)
	if want := srv.BaseURL + "/send/" + srv.path + "/watchface.zpk"; srv.SendURL != want {
		t.Fatalf("SendURL %q, want %q", srv.SendURL, want)
	}
	w := startWaiting(srv)
	resp := get(t, srv.SendURL)
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !bytes.Equal(got, content) {
		t.Fatalf("received %d bytes, want %d", len(got), len(content))
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, `filename="watchface.zpk"`) {
		t.Errorf("Content-Disposition %q does not carry the name", cd)
	}
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop after the download")
	}
}

func TestWithNameEscapesTheName(t *testing.T) {
	for _, name := range []string{"my watch face.zpk", "100%.zpk", "a#b?c.zpk", "quadrante è.zpk", "a;b=c&d.zpk"} {
		t.Run(name, func(t *testing.T) {
			srv := newConfiguredServer(t, "", true)
			sendFile(t, srv, name, []byte(name))
			parsed, err := url.Parse(srv.SendURL)
			if err != nil {
				t.Fatalf("SendURL %q is not a valid URL: %v", srv.SendURL, err)
			}
			if parsed.RawQuery != "" || parsed.Fragment != "" {
				t.Errorf("SendURL %q has a query or a fragment", srv.SendURL)
			}
			if want := "/send/" + srv.path + "/" + name; parsed.Path != want {
				t.Errorf("SendURL path %q, want %q", parsed.Path, want)
			}
			resp, err := http.Get(srv.SendURL)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if string(got) != name {
				t.Errorf("status %d, received %q", resp.StatusCode, got)
			}
		})
	}
}

func TestWithNameOnlyServesTheName(t *testing.T) {
	srv := newConfiguredServer(t, "", true)
	sendFile(t, srv, "watchface.zpk", []byte("content"))
	prefix := srv.BaseURL + "/send/" + srv.path
	for _, suffix := range []string{"/", "/other.zpk", "/watchface.zpk/", "/watchface.zpk/x", "/WATCHFACE.ZPK", "/watchface"} {
		if code := status(t, prefix+suffix); code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", suffix, code)
		}
	}
	// The URL without the name keeps working
	if code := status(t, prefix); code != http.StatusOK {
		t.Errorf("URL without the name: status %d, want 200", code)
	}
}

func TestWithNameFollowsSend(t *testing.T) {
	srv := newConfiguredServer(t, "", true)
	sendFile(t, srv, "first.zpk", []byte("first"))
	first := srv.SendURL
	sendFile(t, srv, "second.zpk", []byte("second"))
	if !strings.HasSuffix(srv.SendURL, "/second.zpk") {
		t.Fatalf("SendURL %q does not end with the new name", srv.SendURL)
	}
	if code := status(t, first); code != http.StatusNotFound {
		t.Errorf("URL of the previous file: status %d, want 404", code)
	}
	resp, err := http.Get(srv.SendURL)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(got) != "second" {
		t.Errorf("received %q", got)
	}
}

func TestSchemeAndWithName(t *testing.T) {
	srv := newConfiguredServer(t, "zpkd1", true)
	sendFile(t, srv, "watchface.zpk", []byte("watchface"))
	host := strings.TrimPrefix(srv.BaseURL, "http://")
	if want := "zpkd1://" + host + "/send/" + srv.path + "/watchface.zpk"; srv.SendURL != want {
		t.Fatalf("SendURL %q, want %q", srv.SendURL, want)
	}
	resp, err := http.Get(httpURL(t, srv, srv.SendURL))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(got) != "watchface" {
		t.Errorf("received %q", got)
	}
}

func TestWithNameDoesNotChangeReceive(t *testing.T) {
	srv := newConfiguredServer(t, "zpkd1", true)
	if err := srv.ReceiveTo(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if want := srv.BaseURL + "/receive/" + srv.path; srv.ReceiveURL != want {
		t.Errorf("ReceiveURL %q, want %q", srv.ReceiveURL, want)
	}
	if code := status(t, srv.ReceiveURL); code != http.StatusOK {
		t.Errorf("upload page: status %d", code)
	}
}
