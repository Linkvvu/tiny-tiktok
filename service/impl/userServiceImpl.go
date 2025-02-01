package impl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"tiktok/dao"
	"tiktok/middleware/jwt"
	"tiktok/service"
	"tiktok/util"

	"gorm.io/gorm"
)

type UserServiceImpl struct{}

func NewUserService() *UserServiceImpl {
	return &UserServiceImpl{}
}

func encrypt(str string) string {
	h := hmac.New(sha256.New, []byte(str))
	sha := hex.EncodeToString(h.Sum([]byte(util.EncryptSecret)))
	return sha
}

func (s *UserServiceImpl) Register(username, password string) (info service.AuthInfo, err error) {
	_, err = dao.GetUserByUsername(username)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("An unexpected error occurred, detail:", err.Error())
			err = fmt.Errorf("%w: %w", util.ErrUnknown, err)
			return
		}
		// do register
		pw := encrypt(password)
		user_dao := dao.User{Username: username, Password: pw}
		err = dao.PersistUser(&user_dao)
		if err != nil {
			log.Println("An unexpected error occurred, detail:", err.Error())
			err = fmt.Errorf("%w: %w", util.ErrUnknown, err)
			return
		}

		var token string
		token, err = jwt.NewToken(
			strconv.FormatInt(user_dao.Id, 10),
		)
		if err != nil {
			return service.AuthInfo{}, util.ErrInternalService
		}
		info.Fill(user_dao.Id, user_dao.Username, token)
		return
	}
	err = util.ErrAccountExisted
	return
}

func (s *UserServiceImpl) Login(username, password string) (service.AuthInfo, error) {
	user_dao, err := dao.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.AuthInfo{}, util.ErrSignFailed
		} else {
			return service.AuthInfo{}, util.ErrUnknown
		}
	}
	pw := encrypt(password)
	if pw != user_dao.Password {
		return service.AuthInfo{}, util.ErrSignFailed
	}

	info := service.AuthInfo{}
	var token string
	token, err = jwt.NewToken(
		strconv.FormatInt(user_dao.Id, 10),
	)
	if err != nil {
		return service.AuthInfo{}, util.ErrInternalService
	}
	info.Fill(user_dao.Id, user_dao.Username, token)
	return info, nil
}

func (s *UserServiceImpl) GetInfo(author_id, user_id int64) (service.UserInfo, error) {
	user_dao, err := dao.GetUserById(author_id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.UserInfo{}, util.ErrInvalidParam
		} else {
			return service.UserInfo{}, util.ErrUnknown
		}
	}
	info := service.BuildUserInfo(user_dao, user_id)
	return info, nil
}
