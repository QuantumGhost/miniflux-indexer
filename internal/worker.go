package internal

import (
	"context"
	"github.com/huichen/sego"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/morikuni/failure"
	"github.com/rs/zerolog/log"
	"runtime"
	"strings"
	"sync"
	"time"
)

const batchSize = 200

type Worker struct {
	parentCtx  context.Context
	config     Config
	infoDB     *pgxpool.Pool
	minifluxDB *pgxpool.Pool
}

func NewWorker(ctx context.Context, config Config) *Worker {
	return &Worker{parentCtx: ctx, config: config}
}

func (w *Worker) indexEntry(ctx context.Context, entry Entry, segmenter *sego.Segmenter) error {
	const (
		updateDocVectorSQL = `UPDATE entries SET document_vector = to_tsvector('simple', ?) WHERE id = ?`
		insertIndexInfoSQL = `INSERT INTO index_info (id, meta) VALUES (?, ?)`
	)
	minifluxDB := w.minifluxDB
	seg := segmenter.Segment([]byte(entry.Content))
	_, err := minifluxDB.Exec(ctx, updateDocVectorSQL, strings.Join(sego.SegmentsToSlice(seg, false), " "), entry.ID)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	_, err = w.infoDB.Exec(ctx, insertIndexInfoSQL, entry.ID, pgtype.JSON{})
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	return nil
}

func (w *Worker) indexWorker(jobChan <-chan Entry) {
	var segmenter sego.Segmenter
	segmenter.LoadDictionary(".")
	for {
		select {
		case entry := <-jobChan:
			ctx, _ := context.WithTimeout(w.parentCtx, defaultTimeout)
			err := w.indexEntry(ctx, entry, &segmenter)
			if err != nil {
				log.Error().Err(err).Msg("error while index entry")
			}
		case <-w.parentCtx.Done():
			return
		}
	}
}

func (w *Worker) findEntriesToIndex(jobChan chan<- Entry) error {
	const maxIDQuery = "SELECT MAX(id) as max_id FROM index_info"
	var maxIndexedID int64 = 0
	infoDB := w.infoDB
	if !w.config.Reindex {
		ctx, cancel := context.WithTimeout(w.parentCtx, defaultTimeout)
		defer cancel()
		row, err := infoDB.Query(ctx, maxIDQuery)
		if err != nil {
			return failure.MarkUnexpected(err)
		}
		err = row.Scan(&maxIndexedID)
		if err != nil {
			return failure.MarkUnexpected(err)
		}
	}

	minifluxDB := w.minifluxDB
	for {
		const selectEntryQuery = "SELECT id, title, content FROM entries WHERE id > ? LIMIT ?"
		var entry Entry
		ctx, _ := context.WithTimeout(w.parentCtx, defaultTimeout)
		rows, err := minifluxDB.Query(ctx, selectEntryQuery, maxIndexedID, batchSize)
		if err != nil {
			log.Error().Err(failure.MarkUnexpected(err)).Msg("error while query entries to index")
		}
		itemCount := 0
		for rows.Next() {
			err = rows.Scan(&entry.ID, &entry.Title, &entry.Content)
			if err != nil {
				log.Error().Err(failure.MarkUnexpected(err)).Msg("error while scan entry")
				continue
			}
			jobChan <- entry
			if entry.ID > maxIndexedID {
				maxIndexedID = entry.ID
			}
			itemCount++
		}
		// # NOTE(QuantumGhost): A full batch, start next batch now
		if itemCount == batchSize {
			select {
			case <-w.parentCtx.Done():
				return nil
			default:
				continue
			}
		}
		select {
		case <-w.parentCtx.Done():
			return nil
		case <-time.After(w.config.ScanInterval):
			continue
		}
	}
}

func (w *Worker) Start() error {
	ctx, cancel := context.WithTimeout(w.parentCtx, defaultTimeout)
	defer cancel()

	db, err := pgxpool.Connect(ctx, w.config.DatabaseURL)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	w.infoDB = db

	ctx, cancel = context.WithTimeout(w.parentCtx, defaultTimeout)
	defer cancel()
	minifluxDB, err := pgxpool.Connect(ctx, w.config.MinifluxDatabaseUrl)
	if err != nil {
		return failure.MarkUnexpected(err)
	}
	w.minifluxDB = minifluxDB

	workerNum := runtime.NumCPU()
	wg := sync.WaitGroup{}
	wg.Add(workerNum)
	jobChan := make(chan Entry, 2*batchSize)
	for i := 0; i < workerNum; i++ {
		go func() {
			w.indexWorker(jobChan)
			wg.Done()
		}()
	}
	wg.Wait()
	return w.findEntriesToIndex(jobChan)
}
