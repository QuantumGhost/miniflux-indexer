package internal

import (
	"github.com/jackc/pgtype"
	"time"
)

type IndexInfo struct {
	ID        int64        `db:"id" json:"id"`
	IndexedAt time.Time    `db:"indexed_at" json:"indexed_at"`
	Meta      pgtype.JSONB `db:"meta" json:"meta"`
}

type Entry struct {
	ID      int64  `db:"id"`
	Title   string `db:"title"`
	Content string `db:"content"`
}
