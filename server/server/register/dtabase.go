package sqlitedb

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/sushiag/go-webrtc-signaling-server/server/server/db"
)

func NewDatabase(filename string) (*db.Queries, *sql.DB) {
	dsn := fmt.Sprintf("file:%s?cache=shared", filename)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatalf("[SERVER] Failed to open DB: %v", err)
	}

	if err := applySchema(conn, "../server/server/db/schema.sql"); err != nil {
		log.Fatalf("[SERVER] Failed to apply schema: %v", err)
	}
	return db.New(conn), conn
}

func applySchema(conn *sql.DB, path string) error {
	schemaSQL, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(schemaSQL))
	return err
}
