package sqlite

import (
	"M3u8Download/utils"
	"database/sql"
	_ "github.com/glebarez/go-sqlite"
	"os"
)

type Sqlite struct {
	Cxt *sql.DB
}

func GetSqliteCxt() (*Sqlite, error) {
	var dbPath = utils.GetDataDir() + "/db.db"
	if utils.IsDockerByCGroup() {
		if !utils.FileExists(dbPath) {
			if !utils.DirExists(utils.GetDataDir()) {
				_ = os.MkdirAll(utils.GetDataDir(), os.ModePerm)
			}
			_ = utils.CopyFile("./data/db.db", dbPath)
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return &Sqlite{Cxt: db}, nil
}
