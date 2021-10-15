package dbops

import (
	"database/sql"
	"fmt"
	"time"
	"video_server/defs"

	uuid "github.com/satori/go.uuid"
)

func AddNewVideo(aid int, name string) (*defs.VideoInfo, error) {
	vid := uuid.NewV4().String()

	t := time.Now()

	ctime := t.Format("Jan 02 2006, 15:04:05")

	stmtIns, err := conn.Prepare("INSERT INTO video_info (id, author_id, name, display_ctime) VALUES(?,?,?,?)")

	if err != nil {
		return nil, err
	}
	_, err = stmtIns.Exec(vid, aid, name, ctime)
	defer stmtIns.Close()
	if err != nil {
		return nil, err
	}
	return &defs.VideoInfo{Id: vid, AuthorId: aid, Name: name, DisplayTime: ctime}, err
}

func DelVideoInfo(vid string) error {
	stmtDel, err := conn.Prepare("DELETE FROM video_info WHERE id=?")
	if err != nil {
		return err
	}
	_, err = stmtDel.Exec(vid)
	defer stmtDel.Close()
	if err != nil {
		return err
	}
	return err
}

func GetVideoInfo(vid string) (*defs.VideoInfo, error) {

	stmtGet, err := conn.Prepare("Select author_id, name, display_ctime FROM video_info where id = ?")

	if err != nil {
		return nil, err
	}
	var author_id int
	var name, display_ctime string
	err = stmtGet.QueryRow(vid).Scan(&author_id, &name, &display_ctime)
	fmt.Println(author_id, name, display_ctime, vid)
	defer stmtGet.Close()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &defs.VideoInfo{Id: vid, AuthorId: author_id, Name: name, DisplayTime: display_ctime}, err
}
