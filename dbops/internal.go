package dbops

import (
	"database/sql"
	"log"
	"strconv"
	"sync"
	"video_server/defs"
)

func InsertSession(sid string, ttl int64, userName string) error {
	ttlstr := strconv.FormatInt(ttl, 10)

	smtmIns, err := conn.Prepare("INSERT INTO sessions (session_id, TTL, login_name) VALUSE(?,?,?)")

	if err != nil {
		return err
	}
	_, err = smtmIns.Exec(sid, ttlstr, userName)
	defer smtmIns.Close()
	if err != nil {
		return err
	}
	return nil
}

func RetieveSession(sid string) (*defs.SimpleSession, error) {
	ss := &defs.SimpleSession{}

	smtmOut, err := conn.Prepare("SELECT TTL, login_name FROM sessions WHERE session_id = ?")

	if err != nil {
		return nil, err
	}

	var TTL, loginName string
	err = smtmOut.QueryRow(sid).Scan(&TTL, &loginName)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if res, err := strconv.ParseInt(TTL, 10, 64); err == nil {
		ss.TTL = res
		ss.UserName = loginName
	} else {
		return nil, err
	}
	return ss, err
}

func RetieveSessions() (*sync.Map, error) {
	m := &sync.Map{}

	smtmOut, err := conn.Prepare("SELECT session_id, TTL, login_name FROM sessions")

	if err != nil {
		return nil, err
	}

	rows, err := smtmOut.Query()
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}
	for rows.Next() {
		var id string
		var ttlstr string
		var loginName string
		if err := rows.Scan(&id, &ttlstr, &loginName); err != nil {
			log.Printf("retrive sessions err: %s", err)
		}

		if ttl, err1 := strconv.ParseInt(ttlstr, 10, 64); err1 == nil {
			ss := &defs.SimpleSession{UserName: loginName, TTL: ttl}
			m.Store(id, ss)
			log.Printf("get session id: %s, ttl: %d \n", id, ss.TTL)
		}
	}
	return m, nil
}

func ReleaseSession(sid string) error {
	stmtDel, err := conn.Prepare("DELETE FROM users WHERE session_id = ?")
	if err != nil {
		log.Print(err)
		return err
	}
	_, err = stmtDel.Exec(sid)
	defer stmtDel.Close()
	if err != nil {
		log.Print(err)
	}
	return err
}
