package qr

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestRenderImage(t *testing.T) {
	img := RenderImage("https://example.com")
	if img == nil {
		t.Fatal("expected non-nil image from RenderImage")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 256 || bounds.Dy() != 256 {
		t.Errorf("expected 256x256 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderStringWhenPiped(t *testing.T) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = stdoutW
	os.Stderr = stderrW

	RenderString("https://example.com", false)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = io.Copy(&stdoutBuf, stdoutR)
	_ = stdoutR.Close()
	_, _ = io.Copy(&stderrBuf, stderrR)
	_ = stderrR.Close()

	if stdoutBuf.Len() != 0 {
		t.Errorf("expected stdout to be empty when piped, got %d bytes: %s", stdoutBuf.Len(), stdoutBuf.String())
	}
	if stderrBuf.Len() == 0 {
		t.Errorf("expected QR code output on stderr when piped, got 0 bytes")
	}
}
