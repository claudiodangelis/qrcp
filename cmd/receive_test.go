package cmd

import (
	"testing"
)

func TestReceiveFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantFileName string
		wantOutput   string
		wantOutDir   string
	}{
		{
			name:         "file flag long",
			args:         []string{"--file", "notes.txt"},
			wantFileName: "notes.txt",
		},
		{
			name:         "file flag short",
			args:         []string{"-f", "notes.txt"},
			wantFileName: "notes.txt",
		},
		{
			name:         "file flag stdout",
			args:         []string{"--file", "-"},
			wantFileName: "-",
		},
		{
			name:       "output flag long",
			args:       []string{"--output", "/tmp/receive"},
			wantOutput: "/tmp/receive",
		},
		{
			name:       "output flag short",
			args:       []string{"-o", "/tmp/receive"},
			wantOutput: "/tmp/receive",
		},
		{
			name:       "output-dir flag",
			args:       []string{"--output-dir", "/tmp/receive"},
			wantOutDir: "/tmp/receive",
		},
		{
			name:         "both file and output-dir flags",
			args:         []string{"--file", "output.txt", "--output-dir", "/tmp/receive"},
			wantFileName: "output.txt",
			wantOutDir:   "/tmp/receive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.Flags.FileName = ""
			app.Flags.Output = ""
			app.Flags.OutputDir = ""

			err := receiveCmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if tt.wantFileName != "" && app.Flags.FileName != tt.wantFileName {
				t.Errorf("expected FileName=%q, got %q", tt.wantFileName, app.Flags.FileName)
			}
			if tt.wantOutput != "" && app.Flags.Output != tt.wantOutput {
				t.Errorf("expected Output=%q, got %q", tt.wantOutput, app.Flags.Output)
			}
			if tt.wantOutDir != "" && app.Flags.OutputDir != tt.wantOutDir {
				t.Errorf("expected OutputDir=%q, got %q", tt.wantOutDir, app.Flags.OutputDir)
			}
		})
	}
}
