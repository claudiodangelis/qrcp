package server

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/claudiodangelis/qrcp/body"
	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/qr"
)

// syncBuffer is a bytes.Buffer safe for concurrent writes
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLog collects what the server logs, including the panics that
// net/http recovers in handlers, and fails the test if any panic is logged
func captureLog(t *testing.T) {
	t.Helper()
	var buf syncBuffer
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		if strings.Contains(buf.String(), "panic") {
			t.Errorf("a handler panicked:\n%s", buf.String())
		}
	})
}

func newTestServer(t *testing.T, keepAlive bool) *Server {
	t.Helper()
	captureLog(t)
	cfg := config.Config{Interface: "any", Bind: "127.0.0.1", KeepAlive: keepAlive}
	srv, err := New(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func sendServer(t *testing.T, content []byte) *Server {
	t.Helper()
	file := filepath.Join(t.TempDir(), "file.bin")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, false)
	srv.Send(body.Body{Filename: "file.bin", Path: file})
	return srv
}

func receiveServer(t *testing.T) (*Server, string) {
	t.Helper()
	srv := newTestServer(t, false)
	dir := t.TempDir()
	if err := srv.ReceiveTo(dir); err != nil {
		t.Fatal(err)
	}
	return srv, dir
}

// stopTimeout is how long tests wait for the server to stop. Shutdown can
// take up to 5 seconds when a client opened a connection it didn't use.
const stopTimeout = 15 * time.Second

// waiter calls srv.Wait in the background
type waiter chan struct{}

func startWaiting(srv *Server) waiter {
	w := make(waiter)
	go func() {
		_ = srv.Wait()
		close(w)
	}()
	return w
}

// stopped reports whether the server stopped within d
func (w waiter) stopped(d time.Duration) bool {
	select {
	case <-w:
		return true
	case <-time.After(d):
		return false
	}
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestSendServesTheFile(t *testing.T) {
	content := bytes.Repeat([]byte("qrcp"), 100000)
	srv := sendServer(t, content)
	w := startWaiting(srv)
	resp := get(t, srv.SendURL)
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !bytes.Equal(got, content) {
		t.Fatalf("received %d bytes, want %d", len(got), len(content))
	}
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop after the download")
	}
}

func TestConcurrentFirstRequests(t *testing.T) {
	// Several requests of a browser arriving at once, before any has the
	// session cookie: exactly one starts the session and is served, and the
	// server neither panics nor stops before it is done.
	content := bytes.Repeat([]byte("x"), 1<<20)
	srv := sendServer(t, content)
	w := startWaiting(srv)
	const n = 8
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		go func() {
			req, _ := http.NewRequest("GET", srv.SendURL, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				codes <- 0
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			codes <- resp.StatusCode
		}()
	}
	served := 0
	for i := 0; i < n; i++ {
		switch code := <-codes; code {
		case http.StatusOK:
			served++
		case http.StatusBadRequest:
			// Another browser, or a request without the session cookie
		case 0:
			// Arrived after the download was done and the server stopped;
			// a handler panic would be caught by captureLog instead
		default:
			t.Errorf("unexpected status %d", code)
		}
	}
	if served != 1 {
		t.Errorf("%d requests were served, want 1", served)
	}
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop after the download")
	}
}

func TestSessionRequestsAreCounted(t *testing.T) {
	srv := newTestServer(t, false)
	id, err := srv.startSessionRequest("")
	if err != nil || id == "" {
		t.Fatalf("first request: %q, %v", id, err)
	}
	if gotID, err := srv.startSessionRequest("wrong"); err != errMismatchingSession || gotID != "" {
		t.Fatalf("wrong cookie: %q, %v", gotID, err)
	}
	if gotID, err := srv.startSessionRequest(id); err != nil || gotID != "" {
		t.Fatalf("second request: %q, %v", gotID, err)
	}
	w := startWaiting(srv)
	srv.endSessionRequest()
	if w.stopped(100 * time.Millisecond) {
		t.Fatal("stopped while a request was still in progress")
	}
	srv.endSessionRequest()
	if !w.stopped(stopTimeout) {
		t.Fatal("did not stop after the last request")
	}
}

func TestOnlyTheRouteOfTheModeIsServed(t *testing.T) {
	// An upload while sending must not reach the upload handler
	send := sendServer(t, []byte("content"))
	w := startWaiting(send)
	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	part, _ := mw.CreateFormFile("files", "a.txt")
	_, _ = part.Write([]byte("hello"))
	_ = mw.Close()
	resp, err := http.Post(strings.Replace(send.SendURL, "/send/", "/receive/", 1), mw.FormDataContentType(), &form)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("upload while sending: status %d", resp.StatusCode)
	}
	send.Shutdown()
	w.stopped(stopTimeout)

	// A download while receiving must not reach the download handler
	recv, _ := receiveServer(t)
	w = startWaiting(recv)
	resp = get(t, strings.Replace(recv.ReceiveURL, "/receive/", "/send/", 1))
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("download while receiving: status %d", resp.StatusCode)
	}
	recv.Shutdown()
	w.stopped(stopTimeout)
}

func TestReceiveSavesTheFiles(t *testing.T) {
	srv, dir := receiveServer(t)
	w := startWaiting(srv)
	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	part, _ := mw.CreateFormFile("files", "photo.jpg")
	_, _ = part.Write([]byte("not really a photo"))
	_ = mw.Close()
	resp, err := http.Post(srv.ReceiveURL, mw.FormDataContentType(), &form)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop after the upload")
	}
	if saved, _ := os.ReadFile(filepath.Join(dir, "photo.jpg")); string(saved) != "not really a photo" {
		t.Errorf("saved %q", saved)
	}
}

func TestMalformedUploadIsAnsweredAndServingContinues(t *testing.T) {
	srv, dir := receiveServer(t)
	w := startWaiting(srv)
	const boundary = "qrcp-test-boundary"
	payload := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"files\"; filename=\"a.txt\"\r\n\r\nhello\r\n" +
		"--" + boundary + "\r\n" +
		"this is not a header\r\n\r\nworld\r\n" +
		"--" + boundary + "--\r\n"
	resp, err := http.Post(srv.ReceiveURL, "multipart/form-data; boundary="+boundary, strings.NewReader(payload))
	if err != nil {
		t.Fatalf("no answer, the handler probably panicked: %v", err)
	}
	answer, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(answer), "Upload error") {
		t.Errorf("status %d, answer %q", resp.StatusCode, answer)
	}
	if w.stopped(200 * time.Millisecond) {
		t.Fatal("the server stopped after a malformed upload")
	}
	// The user can try again
	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	part, _ := mw.CreateFormFile("files", "b.txt")
	_, _ = part.Write([]byte("retry"))
	_ = mw.Close()
	resp, err = http.Post(srv.ReceiveURL, mw.FormDataContentType(), &form)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop after the retried upload")
	}
	if saved, _ := os.ReadFile(filepath.Join(dir, "b.txt")); string(saved) != "retry" {
		t.Errorf("saved %q", saved)
	}
}

func TestSessionStartsOnceUnderConcurrency(t *testing.T) {
	// Unlike TestConcurrentFirstRequests, this always makes the callers
	// overlap: they all wait on the same signal
	srv := newTestServer(t, false)
	const n = 64
	start := make(chan struct{})
	firsts := make(chan bool, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			id, _ := srv.startSessionRequest("")
			firsts <- id != ""
		}()
	}
	close(start)
	wg.Wait()
	close(firsts)
	count := 0
	for first := range firsts {
		if first {
			count++
		}
	}
	if count != 1 {
		t.Errorf("%d callers started the session, want 1", count)
	}
}

func TestSessionStopsTheServerOnce(t *testing.T) {
	srv := newTestServer(t, false)
	id, _ := srv.startSessionRequest("")
	srv.endSessionRequest() // the count drops to zero: stop
	if _, err := srv.startSessionRequest(id); err != nil {
		t.Fatal(err)
	}
	srv.endSessionRequest() // zero again: stopping again has no effect
	select {
	case <-srv.stopped:
	default:
		t.Fatal("the end of the session did not stop the server")
	}
	if err := srv.Wait(); err != nil {
		t.Fatal(err)
	}
}

// brokenWriter fails like a connection the client has closed
type brokenWriter struct{ header http.Header }

func (w *brokenWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
func (w *brokenWriter) WriteHeader(int)           {}

func TestPagesRender(t *testing.T) {
	for _, page := range []*template.Template{uploadPage, donePage} {
		var buf bytes.Buffer
		serveTemplate(page, &buf, struct{ Route, File string }{"/receive/x", "a.txt"})
		if buf.Len() == 0 {
			t.Errorf("%s page is empty", page.Name())
		}
	}
}

func TestWritingToAGoneClientDoesNotPanic(t *testing.T) {
	captureLog(t)
	// The done page after an upload, when the phone already went away
	serveTemplate(donePage, &brokenWriter{}, struct{ Route, File string }{})
	// The QR code of --browser, when the browser went away
	qrHandler(qr.RenderImage("http://example.com"))(&brokenWriter{}, httptest.NewRequest("GET", "/qr", nil))
}

func TestUnreadableOutputDirectory(t *testing.T) {
	// Refused at startup
	srv := newTestServer(t, false)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o300); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()
	if os.Getuid() != 0 { // root can read anyway
		if err := srv.ReceiveTo(dir); err == nil {
			t.Error("ReceiveTo accepted a directory that can't be read")
		}
	}

	// Removed after startup: answered, and serving continues
	srv, dir = receiveServer(t)
	w := startWaiting(srv)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	part, _ := mw.CreateFormFile("files", "a.txt")
	_, _ = part.Write([]byte("hello"))
	_ = mw.Close()
	resp, err := http.Post(srv.ReceiveURL, mw.FormDataContentType(), &form)
	if err != nil {
		t.Fatalf("no answer, the handler probably panicked: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status %d", resp.StatusCode)
	}
	if w.stopped(200 * time.Millisecond) {
		t.Error("the server stopped")
	}
	srv.Shutdown()
	w.stopped(stopTimeout)
}

func TestSendAndReceiveToCanBeCalledAgain(t *testing.T) {
	srv := newTestServer(t, false)
	srv.Send(body.Body{Filename: "a", Path: "a"})
	srv.Send(body.Body{Filename: "b", Path: "b"})
	if srv.body.Path != "b" {
		t.Errorf("body not updated: %v", srv.body)
	}
	recv := newTestServer(t, false)
	first, second := t.TempDir(), t.TempDir()
	if err := recv.ReceiveTo(first); err != nil {
		t.Fatal(err)
	}
	if err := recv.ReceiveTo(second); err != nil {
		t.Fatal(err)
	}
	if recv.outputDir != second {
		t.Errorf("output directory not updated: %s", recv.outputDir)
	}
}

// slowUpload posts a file whose content is written by the caller, so that
// the test controls when the upload finishes
type slowUpload struct {
	pw   *io.PipeWriter
	part io.Writer
	form *multipart.Writer
	done chan error
}

func startSlowUpload(t *testing.T, url, name string) *slowUpload {
	t.Helper()
	pr, pw := io.Pipe()
	form := multipart.NewWriter(pw)
	u := &slowUpload{pw: pw, form: form, done: make(chan error, 1)}
	go func() {
		resp, err := http.Post(url, form.FormDataContentType(), pr)
		if err == nil {
			_ = resp.Body.Close()
		}
		u.done <- err
	}()
	part, err := form.CreateFormFile("files", name)
	if err != nil {
		t.Fatal(err)
	}
	u.part = part
	if _, err := part.Write([]byte("first half, ")); err != nil {
		t.Fatal(err)
	}
	return u
}

// waitHandled waits until the handler is writing the file, i.e. the upload
// is in progress on the server side
func waitHandled(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%s was never created", path)
}

// waitShuttingDown waits until the server stops accepting connections, i.e.
// Wait has started the graceful shutdown
func waitShuttingDown(t *testing.T, srv *Server) {
	t.Helper()
	addr := strings.TrimPrefix(srv.BaseURL, "http://")
	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		// Only a refused connection proves the listener is closed; a timeout
		// on a busy machine doesn't. Windows reports it with its own error
		// code.
		if err != nil && (errors.Is(err, syscall.ECONNREFUSED) || runtime.GOOS == "windows") {
			return
		}
		if err != nil {
			continue
		}
		_ = conn.Close()
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the server never started shutting down")
}

// finish completes the upload and waits for the answer
func (u *slowUpload) finish(t *testing.T) {
	t.Helper()
	_, _ = u.part.Write([]byte("second half"))
	_ = u.form.Close()
	_ = u.pw.Close()
	if err := <-u.done; err != nil {
		t.Fatalf("upload failed: %v", err)
	}
}

func TestInterruptDuringUpload(t *testing.T) {
	// CTRL+C or "q" while an upload is in progress: the upload completes,
	// then qrcp exits. It used to hang, because the handler blocked on
	// stopping a server that was already stopping.
	srv, dir := receiveServer(t)
	w := startWaiting(srv)
	upload := startSlowUpload(t, srv.ReceiveURL, "a.txt")
	waitHandled(t, filepath.Join(dir, "a.txt"))
	srv.Shutdown()
	waitShuttingDown(t, srv)
	upload.finish(t)
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop")
	}
	if saved, _ := os.ReadFile(filepath.Join(dir, "a.txt")); string(saved) != "first half, second half" {
		t.Errorf("saved %q", saved)
	}
}

func TestConcurrentUploads(t *testing.T) {
	// The second of two uploads finishing after the first one stopped the
	// server used to hang qrcp
	srv, dir := receiveServer(t)
	w := startWaiting(srv)
	second := startSlowUpload(t, srv.ReceiveURL, "b.txt")
	waitHandled(t, filepath.Join(dir, "b.txt"))
	first := startSlowUpload(t, srv.ReceiveURL, "a.txt")
	waitHandled(t, filepath.Join(dir, "a.txt"))
	first.finish(t)
	waitShuttingDown(t, srv)
	second.finish(t)
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop")
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if saved, _ := os.ReadFile(filepath.Join(dir, name)); string(saved) != "first half, second half" {
			t.Errorf("%s: saved %q", name, saved)
		}
	}
}

func TestShutdownNeverBlocks(t *testing.T) {
	srv := newTestServer(t, false)
	done := make(chan struct{})
	go func() {
		// Before Wait, several times: it used to block forever
		for i := 0; i < 10; i++ {
			srv.Shutdown()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(stopTimeout):
		t.Fatal("Shutdown blocked")
	}
	// The request made before Wait is kept
	w := startWaiting(srv)
	if !w.stopped(stopTimeout) {
		t.Fatal("Wait did not see the request to stop")
	}
	// After Wait, and a second Wait
	srv.Shutdown()
	if err := srv.Wait(); err != nil {
		t.Fatal(err)
	}
	// A Server returned by New with an error
	(&Server{}).Shutdown()
}

func TestInterruptSignal(t *testing.T) {
	// CTRL+C reaches the server as os.Interrupt
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	srv, dir := receiveServer(t)
	w := startWaiting(srv)
	upload := startSlowUpload(t, srv.ReceiveURL, "a.txt")
	waitHandled(t, filepath.Join(dir, "a.txt"))
	if err := process.Signal(os.Interrupt); err != nil {
		t.Skipf("can't send os.Interrupt on this platform: %v", err)
	}
	waitShuttingDown(t, srv)
	upload.finish(t)
	if !w.stopped(stopTimeout) {
		t.Fatal("the server did not stop")
	}
}
