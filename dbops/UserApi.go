package dbops

import (
	"database/sql"
	"log"
)

func AddUserCredential(loginName string, pwd string) error {
	stmtInsert, err := conn.Prepare("INSERT INTO users (login_name,pwd) VALUES(?,?)")
	if err != nil {
		log.Print(err)
		return err
	}

	_, err = stmtInsert.Exec(loginName, pwd)
	defer stmtInsert.Close()

	if err != nil {
		return err
	}
	return err
}

func GetUserCredential(loginName string) (string, error) {
	stmtGet, err := conn.Prepare("SELECT pwd FROM users WHERE login_name = ?")
	if err != nil {
		log.Print("get error:", err)
		return "", err
	}

	var pwd string
	err = stmtGet.QueryRow(loginName).Scan(&pwd)
	defer stmtGet.Close()
	if err != nil && err != sql.ErrNoRows {
		log.Print(err)
	}
	return pwd, err
}

func DelUserCredential(loginName string, pwd string) error {
	stmtDel, err := conn.Prepare("DELETE FROM users WHERE login_name = ? AND pwd = ?")
	if err != nil {
		log.Print(err)
		return err
	}
	_, err = stmtDel.Exec(loginName, pwd)
	defer stmtDel.Close()
	if err != nil {
		log.Print(err)
	}
	return err
}
