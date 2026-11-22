package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommandReadsStdin(t *testing.T) {
	rootCmd := NewRootCommand()
	var stdout bytes.Buffer
	rootCmd.SetIn(strings.NewReader(`{"id": 1, "created": "2026-11-22"}`))
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--name", "Event"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	want := "model Event {\n  id: int64;\n  created: plainDate;\n}\n"
	if stdout.String() != want {
		t.Fatalf("stdout mismatch:\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestRootCommandReadsFileWithInlineFlag(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "sample.json")
	content := `{"user": {"name": "Alice"}}`
	if err := os.WriteFile(jsonPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp json file: %v", err)
	}

	rootCmd := NewRootCommand()
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--inline", jsonPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	want := "model Root {\n  user: {\n    name: string;\n  };\n}\n"
	if stdout.String() != want {
		t.Fatalf("stdout mismatch:\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestRootCommandMissingFileReturnsError(t *testing.T) {
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{filepath.Join(t.TempDir(), "missing.json")})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
