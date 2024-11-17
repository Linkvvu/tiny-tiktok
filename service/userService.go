package service

import "tiktok/dao"

type UserService interface {
	Register(username, password string) (AuthInfo, error)
	Login(username, password string) (AuthInfo, error)
	GetInfo(id int64) (UserInfo, error)
}

type AuthInfo struct {
	Id       int64  // user id
	Username string // username
	Token    string // token
}

func (i *AuthInfo) Fill(id int64, username, token string) {
	i.Id = id
	i.Username = username
	i.Token = token
}

type UserInfo struct {
	Id          int64  `json:"id"`
	Username    string `json:"name"`
	FollowCnt   uint64 `json:"follow_count"`
	FollowerCnt uint64 `json:"follower_count"`
	IsFollow    bool   `json:"is_follow"`
}

// builds a UserInfo obj
// user: user
// cur_id: self
func BuildUserInfo(user dao.User, cur_user_id int64) UserInfo {
	return UserInfo{
		Id:          user.Id,
		Username:    user.Username,
		FollowCnt:   1000,
		FollowerCnt: 100000000,
		IsFollow:    true,
	}
}
