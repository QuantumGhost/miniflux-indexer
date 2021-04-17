package internal

import (
	"time"
)

const (
	defaultTimeout      = 3 * time.Second
	DefaultScanInterval = 30 * time.Second
	DefaultBatchSize    = 20
)

type Config struct {
	MinifluxDatabaseUrl string
	DatabaseURL         string
	Reindex             bool
	ScanInterval        time.Duration
	BatchSize           int
}

func DefaultConfig() Config {
	return Config{
		Reindex:      false,
		ScanInterval: DefaultScanInterval,
		BatchSize:    DefaultBatchSize,
	}
}
