package dbops

import (
	"log"
)

func AddUserCredential(loginName string, pwd string) error {
	stmtInsert, err := conn.Prepare("INSERT INTO users (login_name,pwd) VALUES(?,?)")
	if err != nil {
		log.Print(err)
		return err
	}

	stmtInsert.Exec(loginName, pwd)
	stmtInsert.Close()
	return err
}

func GetUserCredential(loginName string) (string, error) {
	stmtGet, err := conn.Prepare("SELECT pwd FROM users WHERE login_name = ?")
	if err != nil {
		log.Print("get error:", err)
		return "", err
	}

	var pwd string
	stmtGet.QueryRow(loginName).Scan(&pwd)
	stmtGet.Close()
	return pwd, err
}

func DelUserCredential(loginName string, pwd string) error {
	stmtDel, err := conn.Prepare("DELETE FROM users WHERE login_name = ? AND pwd = ?")
	if err != nil {
		log.Print(err)
		return err
	}
	stmtDel.Exec(loginName, pwd)
	stmtDel.Close()
	return err
}
