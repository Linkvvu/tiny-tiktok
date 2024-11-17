package dao

import (
	"fmt"
)

type User struct {
	Id       int64
	Username string
	Password string
}

func GetUserList() (users []User, err error) {
	err = Db.Find(&users).Error
	if err != nil {
		fmt.Println("failed to get users, detail:", err.Error())
	}
	return
}

func GetUserById(id int64) (User, error) {
	u := User{Id: id}
	err := Db.First(&u).Error
	return u, err
}

func GetUserByUsername(username string) (User, error) {
	u := User{}
	err := Db.First(&u, "username = ?", username).Error
	return u, err
}

func PersistUser(user *User) error {
	return Db.Create(user).Error
}
