package dbops

import (
	"database/sql"
	"github.com/lixiandea/video_server/entity"
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

func DelUserCredential(loginName string) error {
	stmtDel, err := conn.Prepare("DELETE FROM users WHERE login_name = ?")
	if err != nil {
		log.Print(err)
		return err
	}
	_, err = stmtDel.Exec(loginName)
	defer stmtDel.Close()
	if err != nil {
		log.Print(err)
	}
	return err
}

func GetUser(userName string) (*entity.User, error) {
	stmtOut, err := conn.Prepare("SELECT id, pwd FROM users where login_name = ? ")
	if err != nil {
		return nil, err
	}
	var id int
	var pwd string
	err = stmtOut.QueryRow(userName).Scan(&id, &pwd)
	if err != nil {
		return nil, err
	}
	return &entity.User{
		Id:        id,
		LoginName: userName,
		Pwd:       pwd,
	}, nil
}
