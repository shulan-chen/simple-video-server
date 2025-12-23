package dbops

import (
	"database/sql"
	"fmt"
	"log"
	api "video-server/api/defs"
)

func AddUser(userName string, pwd string) error {
	user, err := GetUserByName(userName)
	if user != nil && err == nil {
		return fmt.Errorf("user already exits")
	}

	insert, err := Db.Prepare("insert into users (name,password) values(?,?)")
	if err != nil {
		panic(err)
	}
	_, err = insert.Exec(userName, pwd)
	if err != nil {
		return err
	}
	defer insert.Close()
	return nil
}

func GetUserByName(userName string) (*api.User, error) {
	// 注意：这里返回单个 User，不是 slice
	user := new(api.User)
	err := Db.QueryRow("SELECT * FROM users WHERE name = ?", userName).
		Scan(&user.Id, &user.Username, &user.Password, &user.IsVaild)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", userName)
		}
		log.Println("Query error:", err)
		return nil, err
	}

	return user, nil
}

func DeleteUser(id int, userName string) error {
	if userName == "" {
		return fmt.Errorf("empty userName,can not delete")
	}
	pUser, err := GetUserByName(userName)
	if pUser == nil || err != nil {
		return fmt.Errorf("user not exist")
	}
	insert, err := Db.Prepare("delete from users where id = ? and name = ?")
	if err != nil {
		panic(err)
	}
	_, err = insert.Exec(pUser.Id, userName)
	if err != nil {
		return err
	}
	defer insert.Close()
	return nil
}
