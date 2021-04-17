package main

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/mattn/go-isatty"
	"github.com/morikuni/failure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func importEntries(ctx context.Context, reader io.Reader, db *pgxpool.Pool) error {
	const (
		INSERT_SQL = `INSERT INTO entries (id, title, content) VALUES ($1, $2, $3)`
		BUF_SIZE   = 4 * 1024 * 1024
	)
	log.Info().Msg("start importing")
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 0, BUF_SIZE)
	scanner.Buffer(buffer, BUF_SIZE)
	var entry Entry
	for scanner.Scan() {
		err := json.Unmarshal(scanner.Bytes(), &entry)
		if err != nil {
			return failure.MarkUnexpected(err)
		}
		_, err = db.Exec(ctx, INSERT_SQL, entry.ID, entry.Title, entry.Content)
		if err != nil {
			return failure.MarkUnexpected(err)
		}
	}
	if err := scanner.Err(); err != nil {
		return failure.MarkUnexpected(err)
	}
	return nil
}

type Entry struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func setupLogger() {
	writer := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		if !isatty.IsTerminal(os.Stdout.Fd()) {
			w.NoColor = true
		}
	})
	logger := zerolog.New(writer)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = logger
}

func main() {
	setupLogger()
	var inputFilePath string
	var databaseUrl string

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "database-url",
			Usage:       "database url",
			EnvVars:     []string{"DATABASE_URL"},
			Required:    true,
			Destination: &databaseUrl,
		},
		&cli.StringFlag{
			Name:        "input-file",
			Usage:       "input entry jsonlines dump",
			Required:    true,
			Destination: &inputFilePath,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	app.Action = func(c *cli.Context) error {
		input, err := os.Open(inputFilePath)
		if err != nil {
			return failure.MarkUnexpected(err)
		}
		db, err := pgxpool.Connect(ctx, databaseUrl)
		if err != nil {
			return failure.MarkUnexpected(err)
		}
		return importEntries(ctx, input, db)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT)
		<-sigCh
		cancel()
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("error while importing entries")
	}
}
