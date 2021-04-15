package internal

import "time"

const (
	defaultTimeout      = 3 * time.Second
	DefaultScanInterval = 30 * time.Second
)

type Config struct {
	MinifluxDatabaseUrl string
	DatabaseURL         string
	Reindex             bool
	ScanInterval        time.Duration
}

func DefaultConfig() Config {
	return Config{
		Reindex:      false,
		ScanInterval: DefaultScanInterval,
	}
}
