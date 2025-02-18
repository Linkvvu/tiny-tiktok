package service

type AuthInfo struct {
	Id       uint64 // user id
	Username string // username
	Token    string // token
}

type UserService interface {
	Register(username, password string) (*AuthInfo, error)
	Login(username, password string) (*AuthInfo, error)
	GetInfo(targetUserId, curUserId uint64) (*UserInfo, error)
}

type UserInfo struct {
	Id               uint64 `json:"id"`
	Username         string `json:"username"`
	Nickname         string `json:"nickname"`
	AvatarUrl        string `json:"avatar_url"`
	BackgroundImgUrl string `json:"background_img_url"`
	FollowedCnt      uint64 `json:"followed_count"`
	FollowerCnt      uint64 `json:"follower_count"`
	IsFollowed       bool   `json:"is_followed"`
}
