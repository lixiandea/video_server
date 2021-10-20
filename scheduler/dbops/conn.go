package dbops

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var (
	conn *sql.DB
	err  error
)

func init() {
	conn, err = sql.Open("mysql", "root:Cz05180921.@tcp(localhost:10086)/video_server?charset=utf8")
	if err != nil {
		panic(err)
	}
	err = conn.Ping()
	if err != nil {
		panic(err)
	}
}
