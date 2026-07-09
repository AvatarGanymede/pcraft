package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	cases := []struct {
		in    string
		key   string
		value string
		ok    bool
	}{
		{"FOO=bar", "FOO", "bar", true},
		{"  FOO = bar  ", "FOO", "bar", true},
		{"export FOO=bar", "FOO", "bar", true},
		{`FOO="quoted value"`, "FOO", "quoted value", true},
		{"FOO='single'", "FOO", "single", true},
		{"FOO=", "FOO", "", true},
		{"# comment", "", "", false},
		{"", "", "", false},
		{"NOEQUALS", "", "", false},
		{"=noKey", "", "", false},
	}
	for _, tc := range cases {
		key, value, ok := parseLine(tc.in)
		if ok != tc.ok || key != tc.key || value != tc.value {
			t.Errorf("parseLine(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.in, key, value, ok, tc.key, tc.value, tc.ok)
		}
	}
}

func TestLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv(envHomeDir, home)
	envFile := filepath.Join(home, ".env")
	contents := "# comment\nPCRAFT_TEST_A=alpha\nPCRAFT_TEST_B=\"beta\"\nPCRAFT_TEST_PRESET=fromfile\n"
	if err := os.WriteFile(envFile, []byte(contents), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	// A pre-set variable must not be overwritten by the file.
	t.Setenv("PCRAFT_TEST_PRESET", "fromenv")

	path, applied, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if path != envFile {
		t.Errorf("path = %q, want %q", path, envFile)
	}
	if applied != 2 {
		t.Errorf("applied = %d, want 2 (preset skipped)", applied)
	}
	if got := os.Getenv("PCRAFT_TEST_A"); got != "alpha" {
		t.Errorf("PCRAFT_TEST_A = %q, want alpha", got)
	}
	if got := os.Getenv("PCRAFT_TEST_B"); got != "beta" {
		t.Errorf("PCRAFT_TEST_B = %q, want beta", got)
	}
	if got := os.Getenv("PCRAFT_TEST_PRESET"); got != "fromenv" {
		t.Errorf("PCRAFT_TEST_PRESET = %q, want fromenv (real env wins)", got)
	}
}

func TestLoad_MissingFileIsNotAnError(t *testing.T) {
	t.Setenv(envHomeDir, t.TempDir())
	path, applied, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if applied != 0 {
		t.Errorf("applied = %d, want 0", applied)
	}
	if path == "" {
		t.Error("expected a resolved path even when file is missing")
	}
}
