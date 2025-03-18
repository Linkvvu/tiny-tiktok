package service

type UserService interface {
	RelService
	Register(username, password string) (*AuthInfo, error)
	Login(username, password string) (*AuthInfo, error)
	GetUserInfo(targetUserId, curUserId uint64) (*UserInfo, error)
	GetAllFollowed(targetId, userId int64) ([]UserInfo, error)
	GetAllFollower(targetId, userId int64) ([]UserInfo, error)
}

type AuthInfo struct {
	Id       uint64 // user id
	Username string // username
	Token    string // token
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
