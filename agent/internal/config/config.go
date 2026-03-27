package config

import (
	"os"
	"time"
)

type Config struct {
	Listen          string
	ShutdownTimeout time.Duration
}

func Load() Config {
	listen := os.Getenv("AGENT_LISTEN")
	if listen == "" {
		listen = "/run/iscsi-agent/agent.sock"
	}

	shutdownTimeout := 10 * time.Second
	if v := os.Getenv("AGENT_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			shutdownTimeout = d
		}
	}

	return Config{
		Listen:          listen,
		ShutdownTimeout: shutdownTimeout,
	}
}
