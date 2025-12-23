package dbops

import (
	"fmt"
	"testing"
)

func TestMain(m *testing.M) {
	clearTables()
	m.Run()
}

func clearTables() {
	Db.Exec("truncate users")
	Db.Exec("truncate sessions")
	Db.Exec("truncate video_info")
	Db.Exec("truncate comments")
}

func TestUserWorkFlow(t *testing.T) {
	t.Run("test add user", testAddUser)
	t.Run("test get user", testGetUser)
	t.Run("test delete user", testDeleteUser)
	t.Run("test reget user", testRegetUser)
}

func testAddUser(t *testing.T) {
	err := AddUser("yanghao", "123456")
	if err != nil {
		t.Errorf("Error of AddUser: %v", err)
	}
}

func testGetUser(t *testing.T) {
	pUser, err := GetUserByName("yanghao")
	if pUser == nil || err != nil {
		t.Errorf("Error of GetUser")
		return
	}
	fmt.Println("get the user:", *pUser)
}

func testDeleteUser(t *testing.T) {
	err := DeleteUser(0, "yanghao")
	if err != nil {
		t.Errorf("Error of DeleteUser: %v", err)
	}
}

func testRegetUser(t *testing.T) {
	pUser, err := GetUserByName("yanghao")
	if err != nil {
		t.Errorf("Error of RegetUser: %v", err)
	}

	if pUser != nil {
		t.Errorf("Deleting user test failed")
	}
}
