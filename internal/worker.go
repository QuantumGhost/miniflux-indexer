package internal

import (
	"bytes"
	"code.sajari.com/sego"
	"context"
	"database/sql"
	"github.com/go-shiori/go-readability"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/morikuni/failure"
	"github.com/retarus/whatlanggo"
	"github.com/rs/zerolog/log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Worker struct {
	parentCtx  context.Context
	config     Config
	infoDB     *pgxpool.Pool
	minifluxDB *pgxpool.Pool
}

func NewWorker(ctx context.Context, config Config) *Worker {
	return &Worker{parentCtx: ctx, config: config}
}

func upsertIndexInfo(ctx context.Context, runner pgxtype.Querier, id int64, language string, meta Metadata) error {
	// language: sql
	const upsertIndexInfoSQL = `
INSERT INTO index_info (id, language, meta) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET language = $2, meta = $3
`
	jsonb := pgtype.JSONB{}
	err := jsonb.Set(meta)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	_, err = runner.Exec(ctx, upsertIndexInfoSQL, id, language, jsonb)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	return nil
}

func (w *Worker) indexEntry(ctx context.Context, entry Entry, segmenter *sego.Segmenter) error {
	const (
		// language: sql
		updateDocVectorSQL = `
UPDATE entries
SET
    document_vectors = setweight(to_tsvector('simple', $1), 'A') || setweight(to_tsvector('simple', $2), 'B')
WHERE id = $3`
	)
	minifluxDB := w.minifluxDB
	buffer := bytes.NewBufferString(entry.Content)
	simplified, err := readability.FromReader(buffer, "http://localhost")
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	lang := whatlanggo.DetectLang(simplified.TextContent)
	if lang != whatlanggo.Cmn {
		return upsertIndexInfo(ctx, w.infoDB, entry.ID, lang.Iso6391(), Metadata{})
	}
	titleSeg := segmenter.Segment([]byte(entry.Title))
	contentSeg := segmenter.Segment([]byte(simplified.TextContent))
	_, err = minifluxDB.Exec(
		ctx, updateDocVectorSQL,
		strings.Join(sego.SegmentsToSlice(titleSeg, false), " "),
		strings.Join(sego.SegmentsToSlice(contentSeg, false), " "),
		entry.ID,
	)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	return upsertIndexInfo(ctx, w.infoDB, entry.ID, lang.Iso6391(), Metadata{})
}

func (w *Worker) indexWorker(jobChan <-chan Entry) {
	var segmenter sego.Segmenter
	segmenter.LoadDefaultDictionary()
	for {
		select {
		case entry := <-jobChan:
			ctx, _ := context.WithTimeout(w.parentCtx, defaultTimeout)
			err := w.indexEntry(ctx, entry, &segmenter)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error while index entry")
			}
		case <-w.parentCtx.Done():
			return
		}
	}
}

func (w *Worker) findEntriesToIndex(jobChan chan<- Entry) error {
	const maxIDQuery = "SELECT MAX(id) as max_id FROM index_info"
	var maxIndexedID int64
	infoDB := w.infoDB
	if !w.config.Reindex {
		var maybeMaxID sql.NullInt64
		ctx, cancel := context.WithTimeout(w.parentCtx, defaultTimeout)
		defer cancel()
		row := infoDB.QueryRow(ctx, maxIDQuery)
		err := row.Scan(&maybeMaxID)
		if err != nil && err != sql.ErrNoRows {
			return failure.MarkUnexpected(err)
		}
		if maybeMaxID.Valid {
			maxIndexedID = maybeMaxID.Int64
		}
	}
	log.Info().
		Int64("max_indexed_id", maxIndexedID).
		Msg("max_id read")

	minifluxDB := w.minifluxDB
	for {
		const selectEntryQuery = "SELECT id, title, content FROM entries WHERE id > $1  ORDER BY id LIMIT $2"
		var entry Entry
		rows, err := minifluxDB.Query(w.parentCtx, selectEntryQuery, maxIndexedID, w.config.BatchSize)
		if err != nil {
			log.Error().Stack().Err(failure.MarkUnexpected(err)).Msg("error while query entries to index")
		}

		itemCount := 0
		for rows.Next() {
			err = rows.Scan(&entry.ID, &entry.Title, &entry.Content)
			if err != nil {
				log.Error().Stack().Err(failure.MarkUnexpected(err)).Msg("error while scan entry")
				continue
			}
			log.Debug().Int64("id", entry.ID).Msg("entry loaded")
			jobChan <- entry
			if entry.ID > maxIndexedID {
				maxIndexedID = entry.ID
			}
			itemCount++
		}
		if itemCount == 0 {
			select {
			case <-w.parentCtx.Done():
				return nil
			case <-time.After(w.config.ScanInterval):
				continue
			}
		} else {
			select {
			case <-w.parentCtx.Done():
				return nil
			default:
				continue
			}
		}
	}
}

func (w *Worker) Start() error {
	ctx, _ := context.WithTimeout(w.parentCtx, defaultTimeout)

	infoDB, err := initDB(ctx, w.config.DatabaseURL)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	w.infoDB = infoDB

	ctx, _ = context.WithTimeout(w.parentCtx, defaultTimeout)
	minifluxDB, err := initDB(ctx, w.config.MinifluxDatabaseUrl)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	w.minifluxDB = minifluxDB

	workerNum := runtime.NumCPU()
	wg := sync.WaitGroup{}
	wg.Add(workerNum)
	jobChan := make(chan Entry, w.config.BatchSize)
	for i := 0; i < workerNum; i++ {
		go func() {
			w.indexWorker(jobChan)
			wg.Done()
		}()
	}

	err = w.findEntriesToIndex(jobChan)
	wg.Wait()
	return err
}

func initDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	var level pgx.LogLevel = pgx.LogLevelWarn
	var err error
	levelStr, ok := os.LookupEnv("PGX_LOG_LEVEL")
	if ok {
		level, err = pgx.LogLevelFromString(levelStr)
		if err != nil {
			return nil, failure.MarkUnexpected(err)
		}
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, failure.MarkUnexpected(err)
	}
	cfg.ConnConfig.Logger = zerologadapter.NewLogger(log.Logger)

	cfg.ConnConfig.LogLevel = level
	db, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, failure.MarkUnexpected(err)
	}
	err = db.Ping(ctx)
	if err != nil {
		return nil, failure.MarkUnexpected(err)
	}
	return db, nil
}
