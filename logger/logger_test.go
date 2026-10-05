package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLoggerQuiet(t *testing.T) {
	l := New(true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	l.Print("this should not be printed")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	if buf.Len() != 0 {
		t.Errorf("expected no output when quiet is true, got: %s", buf.String())
	}
}

func TestLoggerPrintWhenPiped(t *testing.T) {
	l := New(false)

	// Make os.Stdout a pipe
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

	l.Print("hello piped logger")

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
		t.Errorf("expected stdout to be empty when piped, got: %s", stdoutBuf.String())
	}
	if !strings.Contains(stderrBuf.String(), "hello piped logger") {
		t.Errorf("expected stderr to contain 'hello piped logger', got: %s", stderrBuf.String())
	}
}
