package dbops

import (
	"database/sql"
	"github.com/lixiandea/video_server/entity"
	"log"
	"time"

	uuid "github.com/satori/go.uuid"
)

func AddNewComment(vid string, aid int, content string) error {
	id := uuid.NewV4().String()
	t := time.Now()

	ctime := t.Format("Jan 02 2006, 15:04:05")
	smtmIns, err := conn.Prepare(
		`INSERT INTO comments 
		(id, author_id, video_id, content, time) VALUES
		(?,?,?,?,?)`)
	if err != nil {
		return err
	}
	_, err = smtmIns.Exec(id, aid, vid, content, ctime)
	defer smtmIns.Close()
	if err != nil {
		return err
	}
	return nil

}

func GetComment(id string) (*entity.Comment, error) {
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

	return &entity.Comment{Id: id, AuthorId: author_id, VideoId: video_id, Content: content, Ctime: ctime}, nil
}

func GetComments(vid string, from, to int) ([]*entity.Comment, error) {
	stmtGet, err := conn.Prepare(`SELECT comments.id, users.login_name, comments.content FROM comments
		INNER JOIN users ON comments.author_id = users.id
		WHERE comments.video_id = ? AND comments.time > FROM_UNIXTIME(?) AND comments.time <= FROM_UNIXTIME(?)
		ORDER BY comments.time DESC`)
	defer stmtGet.Close()
	if err != nil {
		log.Printf("Error get comments: %v", err)
		return nil, err
	}
	var comments []*entity.Comment
	rows, err := stmtGet.Query(vid, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, name, content string
		if err = rows.Scan(&id, &name, &content); err != nil {
			return comments, err
		}
		comments = append(comments, &entity.Comment{Id: id, VideoId: vid, Content: content})
	}
	return comments, nil
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
