package dao

import (
	"fmt"
)

type User struct {
	Id               uint64
	Username         string
	Password         string
	Nickname         string
	AvatarUrl        string
	BackgroundImgUrl string
}

func GetUserList() (users []User, err error) {
	err = Db.Find(&users).Error
	if err != nil {
		fmt.Println("failed to get users, detail:", err.Error())
	}
	return
}

func GetUserById(id uint64) (User, error) {
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
