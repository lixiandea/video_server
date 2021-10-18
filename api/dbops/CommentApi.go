package dbops

import (
	"api/defs"
	"database/sql"
	"time"

	uuid "github.com/satori/go.uuid"
)

func AddNewComment(vid string, aid int, content string) (*defs.Comment, error) {
	id := uuid.NewV4().String()
	t := time.Now()

	ctime := t.Format("Jan 02 2006, 15:04:05")
	smtmIns, err := conn.Prepare(
		`INSERT INTO comments 
		(id, author_id, video_id, content, time) VALUES
		(?,?,?,?,?)`)
	if err != nil {
		return nil, err
	}
	_, err = smtmIns.Exec(id, aid, vid, content, ctime)
	defer smtmIns.Close()
	if err != nil {
		return nil, err
	}
	return &defs.Comment{Id: id, AuthorId: aid, VideoId: vid, Content: content, Ctime: ctime}, nil

}

func GetComments(id string) (*defs.Comment, error) {
	smtmGet, err := conn.Prepare(
		`SELECT author_id, video_id, content, time FROM comments WHERE id = ?`)
	if err != nil {
		return nil, err
	}
	var author_id int
	var video_id, content, ctime string
	err = smtmGet.QueryRow(id).Scan(&author_id, &video_id, &content, &ctime)
	defer smtmGet.Close()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &defs.Comment{Id: id, AuthorId: author_id, VideoId: video_id, Content: content, Ctime: ctime}, nil
}

func DeleteComment(id string) error {
	stmtDel, err := conn.Prepare("DELETE FROM comments WHERE id=?")
	if err != nil {
		return err
	}
	_, err = stmtDel.Exec(id)
	defer stmtDel.Close()
	if err != nil {
		return err
	}
	return err
}
