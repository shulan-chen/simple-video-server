package dbops

import (
	"fmt"
	"testing"
	"time"
)

var test_video_id string

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

/* func TestUserWorkFlow(t *testing.T) {
	t.Run("test add user", testAddUser)
	t.Run("test get user", testGetUser)
	t.Run("test delete user", testDeleteUser)
	t.Run("test reget user", testRegetUser)
} */

func testAddUser(t *testing.T) {
	_, err := AddUser("yanghao", "123456")
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
	if err == nil && pUser != nil {
		t.Errorf("Deleting user test failed")
	}

}

// Video work flow test
/* func TestVideoWorkFlow(t *testing.T) {
	t.Run("test add user", testAddUser)
	t.Run("test add video", testAddVideoInfo)
	t.Run("test get video", testGetVideoInfo)
	t.Run("test delete video", testDeleteVideoInfo)
	t.Run("test reget video", testRegetVideoInfo)
} */

func testAddVideoInfo(t *testing.T) {
	videoInfo, err := AddNewVideo(1, "test_video")
	if err != nil {
		t.Errorf("Error of AddUser: %v", err)
	}
	fmt.Println("add the video info:", *videoInfo)
	test_video_id = videoInfo.Vid
}

func testGetVideoInfo(t *testing.T) {
	videoInfo, err := GetVideoInfo(test_video_id)
	if videoInfo == nil || err != nil {
		t.Errorf("Error of GetVideoInfo")
		return
	}
	fmt.Println("get the video info:", *videoInfo)
}

func testDeleteVideoInfo(t *testing.T) {
	err := DeleteVideoInfo(test_video_id)
	if err != nil {
		t.Errorf("Error of DeleteVideoInfo: %v", err)
	}
}

func testRegetVideoInfo(t *testing.T) {
	videoInfo, err := GetVideoInfo(test_video_id)
	if err == nil && videoInfo != nil {
		t.Errorf("Deleting user test failed")
	}

}

// test comment work flow
func TestCommentsWorkFlow(t *testing.T) {
	t.Run("test add user", testAddUser)
	t.Run("test add video", testAddVideoInfo)
	t.Run("test add comment", testAddComments)
	t.Run("test list comment", testListComments)
}

func testAddComments(t *testing.T) {
	err := InsertNewComments(test_video_id, 1, "this is a test comment")
	if err != nil {
		t.Errorf("Error of AddNewComment: %v", err)
	}
}

func testListComments(t *testing.T) {
	from := time.Now().Add(-time.Hour * 24 * 7)
	to := time.Now()
	commentDTOs, err := ListComments(test_video_id, from, to)
	if err != nil {
		t.Errorf("Error of ListComments: %v", err)
	}
	for _, comment := range commentDTOs {
		fmt.Println("list comment:", *comment)
	}
}

func TestSessionWorkFlow(t *testing.T) {
	t.Run("test add user", testAddUser)
	t.Run("test add session", testAddSession)
	t.Run("test load session", testLoadSession)
}
func testAddSession(t *testing.T) {
	err := InsertNewSession("test_session_id", 1, "yanghao", time.Now().Add(time.Hour*24).Unix())
	if err != nil {
		t.Errorf("Error of AddNewSession: %v", err)
	}
}

func testLoadSession(t *testing.T) {
	sessions, err := LoadSessionsFromDB()
	if err != nil {
		t.Errorf("Error of LoadSessionsFromDB: %v", err)
	}
	for _, session := range sessions {
		fmt.Println("load session:", session)
	}
}

func testLoadOneSession(t *testing.T) {
	session, err := LoadOneSessionFromDB("test_session_id")
	if err != nil {
		t.Errorf("Error of LoadOneSessionFromDB: %v", err)
	}
	fmt.Println("load one session:", *session)
}
