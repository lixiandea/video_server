package dbops

import "testing"

func TruncAllTables() {
	conn.Exec("truncate users")
	conn.Exec("truncate video_info")
	conn.Exec("truncate comment")
	// conn.Exec("truncate sessions")
}
func TestMain(m *testing.M) {
	TruncAllTables()
	m.Run()
	TruncAllTables()
}

func TestUserDBWorkflow(t *testing.T) {
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
	login_name, pwd := "lxd", "qwer1234"
	err := DelUserCredential(login_name, pwd)
	if err != nil {
		t.Errorf("Get user fail: %v", err)
	}
	p, err := GetUserCredential(login_name)
	if p != "" || err != nil {
		t.Fail()
	}
}
