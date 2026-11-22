package server

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/claudiodangelis/qrcp/body"
)

func TestSendHEADDoesNotStartSession(t *testing.T) {
	for _, tc := range []struct {
		name      string
		keepAlive bool
		userAgent string
	}{
		{"browser", false, "Mozilla/5.0"},
		{"browser-keep-alive", true, "Mozilla/5.0"},
		{"curl", false, "curl/8.0"},
		{"curl-keep-alive", true, "curl/8.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte("HEAD is metadata, not the transfer.\n")
			file := filepath.Join(t.TempDir(), "preview.txt")
			if err := os.WriteFile(file, content, 0o644); err != nil {
				t.Fatal(err)
			}
			srv := newTestServer(t, tc.keepAlive)
			srv.Send(body.Body{Filename: "preview.txt", Path: file, DeleteAfterTransfer: true})
			w := startWaiting(srv)
			t.Cleanup(func() {
				srv.Shutdown()
				if !w.stopped(stopTimeout) {
					t.Error("server did not stop during cleanup")
				}
			})
			for _, cookie := range []string{"", "preview-cookie", ""} {
				req := httptest.NewRequest(http.MethodHead, srv.SendURL, nil)
				req.Header.Set("User-Agent", tc.userAgent)
				if cookie != "" {
					req.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
				}
				rr := httptest.NewRecorder()
				srv.sendHandler(rr, req)
				if rr.Code != http.StatusOK || rr.Body.Len() != 0 {
					t.Fatalf("HEAD status/body: %d/%d", rr.Code, rr.Body.Len())
				}
				if rr.Result().ContentLength != int64(len(content)) || rr.Header().Get("Content-Disposition") == "" {
					t.Fatalf("HEAD lost download metadata: %v", rr.Header())
				}
				if len(rr.Result().Cookies()) != 0 {
					t.Fatal("HEAD claimed the download session with a cookie")
				}
				srv.mu.Lock()
				id, active := srv.sessionID, srv.activeRequests
				srv.mu.Unlock()
				if id != "" || active != 0 {
					t.Fatalf("HEAD changed session state: %q/%d", id, active)
				}
				select {
				case <-srv.stopped:
					t.Fatal("HEAD stopped the server before a download")
				default:
				}
				if _, err := os.Stat(file); err != nil {
					t.Fatalf("HEAD removed the temporary body: %v", err)
				}
			}
			// A real HTTP GET must still acquire and complete the download.
			req, err := http.NewRequest(http.MethodGet, srv.SendURL, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("User-Agent", "Mozilla/5.0")
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil || resp.StatusCode != http.StatusOK || !bytes.Equal(got, content) {
				t.Fatalf("GET status/body/error: %d/%q/%v", resp.StatusCode, got, err)
			}
			if tc.keepAlive {
				select {
				case <-srv.stopped:
					t.Fatal("keep-alive GET stopped the server")
				default:
				}
			} else {
				if len(resp.Cookies()) != 1 || resp.Cookies()[0].Name != cookieName {
					t.Fatalf("GET did not acquire the session: %v", resp.Cookies())
				}
				if !w.stopped(stopTimeout) {
					t.Fatal("server did not stop after the actual download")
				}
				if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("temporary body was not removed after GET: %v", err)
				}
			}
		})
	}
}

func TestSendHEADPreservesExistingSession(t *testing.T) {
	srv := sendServer(t, []byte("content"))
	id, err := srv.startSessionRequest("")
	if err != nil {
		t.Fatal(err)
	}
	w := startWaiting(srv)
	t.Cleanup(func() {
		srv.Shutdown()
		if !w.stopped(stopTimeout) {
			t.Error("server did not stop during cleanup")
		}
	})
	for _, cookie := range []string{"", "wrong", id} {
		req := httptest.NewRequest(http.MethodHead, srv.SendURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
		}
		rr := httptest.NewRecorder()
		srv.sendHandler(rr, req)
		want := http.StatusBadRequest
		if cookie == id {
			want = http.StatusOK
		}
		if rr.Code != want || len(rr.Result().Cookies()) != 0 {
			t.Fatalf("cookie %q: status %d, cookies %v", cookie, rr.Code, rr.Result().Cookies())
		}
		srv.mu.Lock()
		gotID, active := srv.sessionID, srv.activeRequests
		srv.mu.Unlock()
		if gotID != id || active != 1 {
			t.Fatalf("HEAD changed active session: %q/%d", gotID, active)
		}
		select {
		case <-srv.stopped:
			t.Fatal("HEAD ended the existing download")
		default:
		}
	}
	srv.endSessionRequest()
	if !w.stopped(stopTimeout) {
		t.Fatal("existing download did not stop the server when complete")
	}
}

// headHeaderGate holds a metadata response open while a download completes.
type headHeaderGate struct {
	*httptest.ResponseRecorder
	entered chan struct{}
	release chan struct{}
}

func (g *headHeaderGate) WriteHeader(code int) {
	close(g.entered)
	<-g.release
	g.ResponseRecorder.WriteHeader(code)
}

func TestSendHEADDoesNotHoldDownloadOpen(t *testing.T) {
	srv := sendServer(t, []byte("content"))
	id, err := srv.startSessionRequest("")
	if err != nil {
		t.Fatal(err)
	}
	gate := &headHeaderGate{httptest.NewRecorder(), make(chan struct{}), make(chan struct{})}
	var release sync.Once
	done := make(chan struct{})
	t.Cleanup(func() {
		release.Do(func() { close(gate.release) })
		select {
		case <-done:
		case <-time.After(stopTimeout):
			t.Error("HEAD handler did not finish")
		}
		srv.Shutdown()
		if err := srv.Wait(); err != nil {
			t.Error(err)
		}
	})
	req := httptest.NewRequest(http.MethodHead, srv.SendURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.AddCookie(&http.Cookie{Name: cookieName, Value: id})
	go func() {
		defer close(done)
		srv.sendHandler(gate, req)
	}()
	select {
	case <-gate.entered:
	case <-time.After(stopTimeout):
		t.Fatal("HEAD did not reach response headers")
	}
	srv.mu.Lock()
	active := srv.activeRequests
	srv.mu.Unlock()
	if active != 1 {
		t.Errorf("in-flight HEAD counted as a download: %d active", active)
	}
	srv.endSessionRequest()
	select {
	case <-srv.stopped:
	default:
		t.Error("HEAD held the completed download open")
	}
	release.Do(func() { close(gate.release) })
	<-done
	if gate.Code != http.StatusOK || gate.Body.Len() != 0 {
		t.Errorf("HEAD status/body: %d/%d", gate.Code, gate.Body.Len())
	}
}

// downloadGateWriter sends the headers, then holds the body until released.
type downloadGateWriter struct {
	http.ResponseWriter
	release chan struct{}
}

func (w downloadGateWriter) Write(p []byte) (int, error) {
	w.ResponseWriter.(http.Flusher).Flush()
	<-w.release
	return w.ResponseWriter.Write(p)
}

func TestSendHEADDuringParallelRanges(t *testing.T) {
	content := bytes.Repeat([]byte("qrcp"), 1024)
	file := filepath.Join(t.TempDir(), "file.bin")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, false)
	gates := map[string]chan struct{}{
		"":                make(chan struct{}),
		"bytes=0-2047":    make(chan struct{}),
		"bytes=2048-4095": make(chan struct{}),
	}
	handler := srv.sendHandler
	srv.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w = downloadGateWriter{w, gates[r.Header.Get("Range")]}
		}
		handler(w, r)
	}
	srv.Send(body.Body{Filename: "file.bin", Path: file})
	w := startWaiting(srv)
	var responses []*http.Response
	released := map[string]bool{}
	t.Cleanup(func() {
		for key, gate := range gates {
			if !released[key] {
				close(gate)
			}
		}
		for _, resp := range responses {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		srv.Shutdown()
		if !w.stopped(stopTimeout) {
			t.Error("server did not stop during cleanup")
		}
	})
	client := &http.Client{Timeout: 3 * time.Second}
	var cookie *http.Cookie
	for _, key := range []string{"", "bytes=0-2047", "bytes=2048-4095"} {
		req, err := http.NewRequest(http.MethodGet, srv.SendURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		if key != "" {
			req.Header.Set("Range", key)
			req.AddCookie(cookie)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		responses = append(responses, resp)
		want := http.StatusPartialContent
		if key == "" {
			want = http.StatusOK
			cookies := resp.Cookies()
			if len(cookies) != 1 || cookies[0].Name != cookieName {
				t.Fatalf("first GET did not get a session cookie: %v", cookies)
			}
			cookie = cookies[0]
		}
		if resp.StatusCode != want {
			t.Fatalf("GET %q: status %d, want %d", key, resp.StatusCode, want)
		}
	}
	// All three real HTTP downloads are now in progress.
	req, err := http.NewRequest(http.MethodHead, srv.SendURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.AddCookie(cookie)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK || len(got) != 0 || resp.ContentLength != int64(len(content)) {
		t.Fatalf("HEAD status/body/length/error: %d/%d/%d/%v", resp.StatusCode, len(got), resp.ContentLength, err)
	}
	for i, key := range []string{"", "bytes=0-2047", "bytes=2048-4095"} {
		close(gates[key])
		released[key] = true
		got, err := io.ReadAll(responses[i].Body)
		_ = responses[i].Body.Close()
		want := content
		switch i {
		case 1:
			want = content[:2048]
		case 2:
			want = content[2048:]
		}
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("GET %q body/error: %d/%v", key, len(got), err)
		}
		if i < 2 {
			select {
			case <-srv.stopped:
				t.Fatal("server stopped before the other ranges completed")
			default:
			}
		}
	}
	if !w.stopped(stopTimeout) {
		t.Fatal("server did not stop after all download ranges completed")
	}
}
