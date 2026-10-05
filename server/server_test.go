package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/claudiodangelis/qrcp/config"
)

func TestServerFileName(t *testing.T) {
	s := &Server{}

	// Empty name
	if err := s.FileName("", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.fileName != "" {
		t.Errorf("expected empty fileName, got %q", s.fileName)
	}

	// Stdout
	if err := s.FileName("-", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.fileName != "-" {
		t.Errorf("expected '-', got %q", s.fileName)
	}

	// Relative name with dir
	if err := s.FileName("notes.txt", "/tmp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.fileName != filepath.Join("/tmp", "notes.txt") {
		t.Errorf("expected %q, got %q", filepath.Join("/tmp", "notes.txt"), s.fileName)
	}

	// Relative name without dir
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := s.FileName("notes.txt", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.fileName != filepath.Join(cwd, "notes.txt") {
		t.Errorf("expected %q, got %q", filepath.Join(cwd, "notes.txt"), s.fileName)
	}

	// Absolute name with dir
	absPath := filepath.Join(cwd, "custom_abs.txt")
	if err := s.FileName(absPath, "/other/dir"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.fileName != absPath {
		t.Errorf("expected %q, got %q", absPath, s.fileName)
	}
}

func sendMultipartUpload(targetURL string, files map[string]string) (*http.Response, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for filename, content := range files {
		part, err := writer.CreateFormFile("files[]", filename)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(part, content); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

func TestServerReceiveCustomFileName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qrcp_test_recv_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := config.Config{
		Interface: "any",
		Bind:      "127.0.0.1",
		Port:      0,
	}
	srv, err := New(&cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := srv.ReceiveTo(tmpDir); err != nil {
		t.Fatalf("ReceiveTo failed: %v", err)
	}
	if err := srv.FileName("saved_note.txt", tmpDir); err != nil {
		t.Fatalf("FileName failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.Wait()
	}()

	resp, err := sendMultipartUpload(srv.ReceiveURL, map[string]string{
		"uploaded_name.txt": "payload for custom note",
	})
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server wait error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}

	expectedFile := filepath.Join(tmpDir, "saved_note.txt")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read expected file %s: %v", expectedFile, err)
	}
	if string(content) != "payload for custom note" {
		t.Errorf("expected file content %q, got %q", "payload for custom note", string(content))
	}
}

func TestServerReceiveMultipleFilesCustomFileName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qrcp_test_multi_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := config.Config{
		Interface: "any",
		Bind:      "127.0.0.1",
		Port:      0,
	}
	srv, err := New(&cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := srv.ReceiveTo(tmpDir); err != nil {
		t.Fatalf("ReceiveTo failed: %v", err)
	}
	if err := srv.FileName("document.txt", tmpDir); err != nil {
		t.Fatalf("FileName failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.Wait()
	}()

	// Send 2 files
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	p1, _ := writer.CreateFormFile("files[]", "file1.txt")
	_, _ = io.WriteString(p1, "first file content")
	p2, _ := writer.CreateFormFile("files[]", "file2.txt")
	_, _ = io.WriteString(p2, "second file content")
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.ReceiveURL, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server wait error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}

	f1 := filepath.Join(tmpDir, "document.txt")
	f2 := filepath.Join(tmpDir, "document(1).txt")

	c1, err := os.ReadFile(f1)
	if err != nil {
		t.Fatalf("failed to read %s: %v", f1, err)
	}
	if string(c1) != "first file content" {
		t.Errorf("expected %q, got %q", "first file content", string(c1))
	}

	c2, err := os.ReadFile(f2)
	if err != nil {
		t.Fatalf("failed to read %s: %v", f2, err)
	}
	if string(c2) != "second file content" {
		t.Errorf("expected %q, got %q", "second file content", string(c2))
	}
}

func TestServerReceiveToStdout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qrcp_test_stdout_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := config.Config{
		Interface: "any",
		Bind:      "127.0.0.1",
		Port:      0,
	}
	srv, err := New(&cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := srv.ReceiveTo(tmpDir); err != nil {
		t.Fatalf("ReceiveTo failed: %v", err)
	}
	if err := srv.FileName("-", ""); err != nil {
		t.Fatalf("FileName failed: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan error, 1)
	go func() {
		done <- srv.Wait()
	}()

	resp, err := sendMultipartUpload(srv.ReceiveURL, map[string]string{
		"stream.txt": "transferred payload directly to stdout",
	})
	if err != nil {
		_ = w.Close()
		os.Stdout = oldStdout
		t.Fatalf("upload failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server wait error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}

	_ = w.Close()
	os.Stdout = oldStdout

	var captured bytes.Buffer
	_, _ = io.Copy(&captured, r)
	_ = r.Close()

	if !strings.Contains(captured.String(), "transferred payload directly to stdout") {
		t.Errorf("expected stdout to contain payload, got: %q", captured.String())
	}
}
