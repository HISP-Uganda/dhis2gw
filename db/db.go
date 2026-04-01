package db

import (
	"dhis2gw/config"
	"sync/atomic"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //import postgres
)

var dbConn atomic.Pointer[sqlx.DB]

// ConnectDB ...
func ConnectDB(dataSourceName string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Init() (*sqlx.DB, error) {
	cfg := config.MustGet().Config
	db, err := ConnectDB(cfg.Database.URI)
	if err != nil {
		return nil, err
	}
	if old := dbConn.Swap(db); old != nil && old != db {
		_ = old.Close()
	}
	return db, nil
}

// GetDB ...
func GetDB() *sqlx.DB {
	return dbConn.Load()
}
