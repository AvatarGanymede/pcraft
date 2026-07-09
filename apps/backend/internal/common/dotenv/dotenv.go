// Package dotenv loads environment variables from a `.env` file living under
// the pcraft home directory (default `~/.pcraft/.env`). It is intentionally
// dependency-free and forgiving: a missing file is not an error, and variables
// already present in the real process environment are never overwritten — the
// real environment always wins, mirroring the profiles precedence rule.
//
// This is how optional integration credentials (JNPM, Lark, admin email, …)
// are supplied without committing secrets to the repo. See `.env.example` at
// the repo root for the documented fields.
package dotenv

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// homeSubdir is the pcraft home directory name under the user's home when
// PCRAFT_HOME_DIR is not set. Kept in sync with common/config.
const homeSubdir = ".pcraft"

// envHomeDir is the explicit home-dir override env var (e.g. a Docker/K8s data
// dir, or <repo>/.pcraft-dev during local development).
const envHomeDir = "PCRAFT_HOME_DIR"

// Load reads `<home>/.env` and injects each `KEY=VALUE` pair into the process
// environment, skipping any key that is already set. It returns the resolved
// file path and the number of variables applied. A missing file yields
// (path, 0, nil).
func Load() (path string, applied int, err error) {
	path = filepath.Join(resolveHomeDir(), ".env")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, 0, nil
		}
		return path, 0, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if setErr := os.Setenv(key, value); setErr == nil {
			applied++
		}
	}
	return path, applied, scanner.Err()
}

// parseLine parses a single `.env` line into a key/value pair. It ignores
// blank lines and `#` comments, tolerates an optional leading `export `, splits
// on the first `=`, and strips a single layer of matching surrounding quotes
// from the value.
func parseLine(raw string) (key, value string, ok bool) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	idx := strings.IndexByte(line, '=')
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	value = unquote(value)
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

// unquote strips one layer of matching single or double quotes.
func unquote(v string) string {
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

// resolveHomeDir mirrors config.ResolvedHomeDir without importing the config
// package (this runs before config.Load): PCRAFT_HOME_DIR (tilde-expanded) or
// ~/.pcraft.
func resolveHomeDir() string {
	if v := strings.TrimSpace(os.Getenv(envHomeDir)); v != "" {
		return expandTilde(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return homeSubdir
	}
	return filepath.Join(home, homeSubdir)
}

// expandTilde expands a leading "~/" to the user's home directory.
func expandTilde(p string) string {
	if !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[2:])
}
