package db

import (
	"dhis2gw/config"


	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //import postgres
)

var db *sqlx.DB

func init() {
	psqlInfo := config.DHIS2GWConf.Database.URI
	//
	var err error
	db, err = ConnectDB(psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
}

// ConnectDB ...
func ConnectDB(dataSourceName string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	return db, nil
}

// GetDB ...
func GetDB() *sqlx.DB {
	return db
}
