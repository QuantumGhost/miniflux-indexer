package main

import (
	"context"
	"fmt"
	"github.com/QuantumGhost/miniflux-indexer/internal"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
	"os/signal"
)

const (
	defaultLogLevel  = "info"
	defaultLogFormat = "auto"
)

func envName(name string) string {
	return "INDEXER_" + name
}

func startCmd() *cli.Command {
	config := internal.DefaultConfig()
	cmd := cli.Command{
		Name:  "start",
		Usage: "start indexer",
	}
	cmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "database-url",
			Usage:       "PostgresQL Database URL for index information",
			EnvVars:     []string{"DATABASE_URL"},
			Destination: &config.DatabaseURL,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "miniflux-database-url",
			Usage:       "PostgresQL Database URL for index information",
			EnvVars:     []string{"MINIFLUX_DATABASE_URL"},
			Destination: &config.MinifluxDatabaseUrl,
			Required:    true,
		},
		&cli.BoolFlag{
			Name:        "reindex",
			Usage:       "reindex all entries for index info, set to true to reindex all entries",
			EnvVars:     []string{envName("REINDEX")},
			Destination: &config.Reindex,
			Required:    false,
		},
		&cli.DurationFlag{
			Name:        "scan-interval",
			Usage:       "interval of worker to scan new (unindexed) entries from database",
			EnvVars:     []string{envName("SCAN_INTERVAL")},
			Destination: &config.ScanInterval,
			Value:       internal.DefaultScanInterval,
			Required:    false,
		},
	}

	cmd.Action = func(c *cli.Context) error {
		worker := internal.NewWorker(c.Context, config)
		return worker.Start()
	}
	return &cmd
}

func main() {
	logLevel := defaultLogLevel
	logFormat := defaultLogFormat

	ctx, cancel := context.WithCancel(context.Background())
	app := cli.NewApp()
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(c.App.Writer, "%v\n%s", c.App.Name, internal.GetBuildInfo())
	}
	app.Usage = "miniflux-indexer"
	app.Version = internal.GetVersion()

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "log level of ars ([trace, debug, info, warn, error])",
			EnvVars:     []string{"LOG_LEVEL"},
			Value:       defaultLogLevel,
			Destination: &logLevel,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Usage:       "log format of ars ([auto, human, json])",
			EnvVars:     []string{"LOG_FORMAT"},
			Value:       defaultLogFormat,
			Destination: &logFormat,
		},
	}

	app.Before = func(c *cli.Context) error {
		return internal.SetUpLogger(logLevel, logFormat)
	}
	app.Commands = []*cli.Command{
		startCmd(),
	}
	app.HideVersion = false

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh)
		<-sigCh
		cancel()
	}()
	err := app.RunContext(ctx, os.Args)
	if err != nil && err != context.Canceled {
		log.Error().Err(err).Msg("error while executing indexer")
		os.Exit(1)
	}
	os.Exit(0)
}
