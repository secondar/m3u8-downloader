package sqlite

import (
	"database/sql"
	_ "github.com/glebarez/go-sqlite"
)

type Sqlite struct {
	Cxt *sql.DB
}

func GetSqliteCxt() (*Sqlite, error) {
	db, err := sql.Open("sqlite", "./data/db.db")
	if err != nil {
		return nil, err
	}
	return &Sqlite{Cxt: db}, nil
}
