package launcher

import (
	"fmt"
	"os"
)

func backendEnv(ports portConfig, logLevel string, debug bool) []string {
	env := os.Environ()
	env = upsertEnv(env, "PCRAFT_SERVER_PORT", fmt.Sprint(ports.BackendPort))
	env = upsertEnv(env, "PCRAFT_AGENT_STANDALONE_PORT", fmt.Sprint(ports.AgentctlPort))
	env = upsertEnv(env, "PCRAFT_DATABASE_PATH", resolveDatabasePath())
	if logLevel != "" {
		env = upsertEnv(env, "PCRAFT_LOG_LEVEL", logLevel)
	}
	if debug {
		env = upsertEnv(env, "PCRAFT_DEBUG_AGENT_MESSAGES", "true")
		env = upsertEnv(env, "PCRAFT_DEBUG_PPROF_ENABLED", "true")
	}
	return env
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, item := range env {
		if len(item) >= len(prefix) && item[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
