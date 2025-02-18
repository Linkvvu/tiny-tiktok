package impl

import (
	"errors"
	"log"
	"strconv"
	"tiktok/dao"
	"tiktok/middleware/jwt"
	uSrv "tiktok/service/user"
	"tiktok/util"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserServiceImpl struct{}

func NewUserService() *UserServiceImpl {
	return &UserServiceImpl{}
}

func (s *UserServiceImpl) Register(username, password string) (*uSrv.AuthInfo, error) {
	_, err := dao.GetUserByUsername(username)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("An unexpected error occurred, detail:", err.Error())
			return nil, util.ErrInternalService
		}

		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("failed to encrypt, detail: %v", err)
			return nil, util.ErrInternalService
		}

		// do register
		user_dao := dao.User{Username: username, Password: string(hashedPwd)}
		err = dao.PersistUser(&user_dao)
		if err != nil {
			log.Println("An unexpected error occurred, detail:", err.Error())
			return nil, util.ErrInternalService
		}

		var token string
		token, err = jwt.NewToken(
			strconv.FormatUint(user_dao.Id, 10),
		)
		if err != nil {
			return nil, util.ErrInternalService
		}

		return &uSrv.AuthInfo{
			Id:       user_dao.Id,
			Username: username,
			Token:    token,
		}, nil
	}
	return nil, util.ErrAccountExisted
}

func (s *UserServiceImpl) Login(username, password string) (*uSrv.AuthInfo, error) {
	user_dao, err := dao.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrSignFailed
		} else {
			return nil, util.ErrInternalService
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user_dao.Password), []byte(password)); err != nil {
		return nil, util.ErrSignFailed
	}

	var token string
	token, err = jwt.NewToken(
		strconv.FormatUint(user_dao.Id, 10),
	)
	if err != nil {
		return nil, util.ErrInternalService
	}

	return &uSrv.AuthInfo{
		Id:       user_dao.Id,
		Username: user_dao.Username,
		Token:    token,
	}, nil
}

func (s *UserServiceImpl) GetInfo(targetUserId, curUserId uint64) (*uSrv.UserInfo, error) {
	user_dao, err := dao.GetUserById(targetUserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.ErrInvalidParam
		} else {
			return nil, util.ErrInternalService
		}
	}
	info := s.buildUserInfo(user_dao, curUserId)
	return &info, nil
}

func (s *UserServiceImpl) buildUserInfo(targetUser dao.User, curUserId uint64) uSrv.UserInfo {
	return uSrv.UserInfo{
		Id:               targetUser.Id,
		Username:         targetUser.Username,
		Nickname:         targetUser.Nickname,
		AvatarUrl:        targetUser.AvatarUrl,
		BackgroundImgUrl: targetUser.BackgroundImgUrl,
		FollowedCnt:      1000,
		FollowerCnt:      100000000,
		IsFollowed:       true,
	}
}
