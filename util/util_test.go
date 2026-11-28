package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TMP", dir)
	t.Setenv("TEMP", dir)
	return dir
}

func TestZipFilesCleansUpAfterInputError(t *testing.T) {
	for _, tc := range []struct {
		name       string
		afterValid bool
	}{
		{name: "first_input"},
		{name: "after_valid_input", afterValid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := zipTempDir(t)
			source := t.TempDir()
			inputs := []string{}
			if tc.afterValid {
				file := filepath.Join(source, "valid.txt")
				if err := os.WriteFile(file, []byte("payload"), 0600); err != nil {
					t.Fatal(err)
				}
				inputs = append(inputs, file)
			}
			missing := filepath.Join(source, "missing.txt")
			inputs = append(inputs, missing)
			name, err := ZipFiles(inputs)
			if err == nil || name != "" || !os.IsNotExist(err) {
				t.Fatalf("name=%q err=%v", name, err)
			}
			entries, readErr := os.ReadDir(tmp)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Errorf("failed zip left %d temporary files: %v", len(entries), entries)
			}
		})
	}
}

func TestZipFilesKeepsCompletedArchive(t *testing.T) {
	tmp := zipTempDir(t)
	source := t.TempDir()
	file := filepath.Join(source, "plain.txt")
	if err := os.WriteFile(file, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := ZipFiles([]string{file})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(name) }()
	if filepath.Dir(name) != tmp || !strings.HasSuffix(name, ".zip") {
		t.Fatalf("unexpected archive name %q", name)
	}
	zr, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	if len(zr.File) != 1 || zr.File[0].Name != "plain.txt" {
		t.Fatalf("entries=%v", zr.File)
	}
	r, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	data, err := io.ReadAll(r)
	if err != nil || string(data) != "content" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary entries=%v err=%v", entries, err)
	}
}

func TestZipFilesCleansUpAfterDirectoryError(t *testing.T) {
	tmp := zipTempDir(t)
	source := t.TempDir()
	if err := os.Symlink(filepath.Join(source, "missing"), filepath.Join(source, "broken")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	name, err := ZipFiles([]string{source})
	if err == nil || name != "" {
		t.Fatalf("name=%q err=%v", name, err)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary entries=%v err=%v", entries, err)
	}
}

func TestZipFilesCleansUpAfterUnreadableDirectoryEntry(t *testing.T) {
	tmp := zipTempDir(t)
	source := t.TempDir()
	file := filepath.Join(source, "unreadable.txt")
	if err := os.WriteFile(file, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0); err != nil {
		t.Skipf("cannot change permissions: %v", err)
	}
	defer func() { _ = os.Chmod(file, 0600) }()
	if f, err := os.Open(file); err == nil {
		_ = f.Close()
		t.Skip("file remains readable despite permission change")
	}
	name, err := ZipFiles([]string{source})
	if err == nil || name != "" {
		t.Fatalf("name=%q err=%v", name, err)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary entries=%v err=%v", entries, err)
	}
}
