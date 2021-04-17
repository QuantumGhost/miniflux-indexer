package internal

import (
	"time"
)

type Metadata struct{}

type IndexInfo struct {
	ID        int64     `db:"id" json:"id"`
	IndexedAt time.Time `db:"indexed_at" json:"indexed_at"`
	Meta      Metadata  `db:"meta"`
}

type Entry struct {
	ID      int64  `db:"id"`
	Title   string `db:"title"`
	Content string `db:"content"`
}
