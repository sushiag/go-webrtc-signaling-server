package sqlitedb

import (
	"database/sql"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/server/lib/server/db"
)

func NewDatabase(datasource string) *db.Queries {
	conn, err := sql.Open("sqlite3", datasource)
	if err != nil {
		log.Fatalf("DATABASE: failed to open sqlite database: ^%v", err)
	}
	return db.New(conn)
}
