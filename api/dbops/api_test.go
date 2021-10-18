package dbops

import (
	"database/sql"
	"testing"
)

var videoID string

var commentId string

func TruncAllTables() {
	conn.Exec("truncate users")
	conn.Exec("truncate video_info")
	conn.Exec("truncate comment")
	// conn.Exec("truncate sessions")
}
func TestMain(m *testing.M) {
	m.Run()
	TruncAllTables()
}

func TestUserDBWorkflow(t *testing.T) {
	TruncAllTables()
	t.Run("Add user", TestAddUser)
	t.Run("Get user", TestGetUser)
	t.Run("Del user", TestDelUser)
}

func TestAddUser(t *testing.T) {
	login_name, pwd := "lxd", "qwer1234"
	err := AddUserCredential(login_name, pwd)
	if err != nil {
		t.Errorf("Add user fail: %v", err)
	}
}

func TestGetUser(t *testing.T) {
	login_name, pwd := "lxd", "qwer1234"
	p, err := GetUserCredential(login_name)
	if p != pwd || err != nil {
		t.Fail()
	}
}

func TestDelUser(t *testing.T) {
	login_name := "lxd"
	err := DelUserCredential(login_name)
	if err != nil {
		t.Errorf("Get user fail: %v", err)
	}
	_, err = GetUserCredential(login_name)
	if err != sql.ErrNoRows {
		t.Fail()
	}
}

func TestVideoWorkflow(t *testing.T) {
	TruncAllTables()
	t.Run("PrepareUser", TestAddUser)
	t.Run("AddVideo", TestAddVideoInfo)
	t.Run("GetVideo", TestGetVideoInfo)
	t.Run("DelVideo", TestDelUser)
}

func TestAddVideoInfo(t *testing.T) {
	vi, err := AddNewVideo(1, "testVideo")
	if err != nil {
		t.Errorf("%v", err)
	}
	videoID = vi.Id
}

func TestGetVideoInfo(t *testing.T) {
	vi, err := GetVideoInfo(videoID)
	if err != nil {
		t.Errorf("%v", err)
	}
	if vi == nil {
		t.Errorf("get no videoInfo")
	} else {
		if vi.Id != videoID {
			t.Errorf("get error id, expect:%s, but get：%s", videoID, vi.Id)
		}
	}

}

func TestDelVideoInfo(t *testing.T) {
	err := DelVideoInfo(videoID)
	if err != nil {
		t.Errorf("%v", err)
	}
	vi, err := GetVideoInfo(videoID)
	if vi != nil || err != nil {
		t.Errorf("fail to delete, err: %v", err)
	}

}

func TestCommentWorkflow(t *testing.T) {
	TruncAllTables()
	t.Run("PrepareUser", TestAddUser)
	t.Run("PrepareVideo", TestAddVideoInfo)
	t.Run("TestAddComment", TestAddComment)
	t.Run("TestGetComment", TestGetComment)
	t.Run("TestDelComment", TestDelComment)
}

func TestAddComment(t *testing.T) {
	comment, err := AddNewComment(videoID, 1, "testVideo")
	if err != nil {
		t.Errorf("%v", err)
	}
	commentId = comment.Id
}

func TestGetComment(t *testing.T) {
	comment, err := GetComments(commentId)
	if err != nil {
		t.Errorf("%v", err)
	}
	if comment == nil {
		t.Errorf("get no commentId")
	} else {
		if comment.Id != commentId {
			t.Errorf("get error id, expect:%s, but get：%s", commentId, comment.Id)
		}
	}

}

func TestDelComment(t *testing.T) {
	err := DeleteComment(commentId)
	if err != nil {
		t.Errorf("%v", err)
	}
	vi, err := GetComments(commentId)
	if vi != nil || err != nil {
		t.Errorf("fail to delete, err: %v", err)
	}

}
