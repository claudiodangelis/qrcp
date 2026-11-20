package util

import (
	"os"
	"testing"
)

func TestOutputIsPipe(t *testing.T) {
	// Nil file check
	if isPipe(nil) {
		t.Errorf("isPipe(nil) expected false, got true")
	}

	// os.Pipe() check
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	if !isPipe(r) {
		t.Errorf("isPipe(reader) expected true for pipe reader, got false")
	}
	if !isPipe(w) {
		t.Errorf("isPipe(writer) expected true for pipe writer, got false")
	}
}
