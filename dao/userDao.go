package dao

import (
	"fmt"
)

type User struct {
	Id               uint64 `redis:"id"`
	Username         string `redis:"username"`
	Password         string `redis:"password"`
	Nickname         string `redis:"nickname"`
	AvatarUrl        string `redis:"avatar_url"`
	BackgroundImgUrl string `redis:"background_url"`
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
