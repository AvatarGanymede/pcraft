package launcher

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultBackendPort  = 38429
	defaultAgentctlPort = 39429

	healthTimeoutReleaseMS = 45000
	randomPortMin          = 10000
	randomPortMax          = 60000
)

func resolveHomeDir() string {
	if v := strings.TrimSpace(os.Getenv("PCRAFT_HOME_DIR")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pcraft"
	}
	return filepath.Join(home, ".pcraft")
}

func resolveDataDir() string {
	return filepath.Join(resolveHomeDir(), "data")
}

func resolveDatabasePath() string {
	if v := strings.TrimSpace(os.Getenv("PCRAFT_DATABASE_PATH")); v != "" {
		return v
	}
	return filepath.Join(resolveDataDir(), "pcraft.db")
}
