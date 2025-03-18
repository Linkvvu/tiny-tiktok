package impl

import (
	"errors"
	"log"
	"math/rand/v2"
	"strconv"
	"tiktok/dao"
	"tiktok/middleware/cache"
	"tiktok/middleware/jwt"
	"tiktok/pkg"
	uSrv "tiktok/service/user"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserServiceImpl struct {
	uSrv.RelService
}

func NewUserService(relSrv uSrv.RelService) *UserServiceImpl {
	return &UserServiceImpl{
		RelService: relSrv,
	}
}

func (s *UserServiceImpl) Register(username, password string) (*uSrv.AuthInfo, error) {
	_, err := dao.GetUserByUsername(username)
	if err == nil {
		return nil, pkg.NewError(pkg.ErrAccountExisted, nil)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Println("An unexpected error occurred, detail:", err.Error())
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to encrypt, detail: %v", err)
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	// do register
	user_dao := dao.User{Username: username, Password: string(hashedPwd)}
	err = dao.PersistUser(&user_dao)
	if err != nil {
		log.Println("An unexpected error occurred, detail:", err.Error())
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	var token string
	token, err = jwt.NewToken(
		strconv.FormatUint(user_dao.Id, 10),
	)
	if err != nil {
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	return &uSrv.AuthInfo{
		Id:       user_dao.Id,
		Username: username,
		Token:    token,
	}, nil
}

func (s *UserServiceImpl) Login(username, password string) (*uSrv.AuthInfo, error) {
	user_dao, err := dao.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkg.NewError(pkg.ErrUnmatchedPwd, nil)
		} else {
			return nil, pkg.NewError(pkg.ErrInternal, err)
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user_dao.Password), []byte(password)); err != nil {
		return nil, pkg.NewError(pkg.ErrUnmatchedPwd, err)
	}

	var token string
	token, err = jwt.NewToken(strconv.FormatUint(user_dao.Id, 10))
	if err != nil {
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	return &uSrv.AuthInfo{
		Id:       user_dao.Id,
		Username: user_dao.Username,
		Token:    token,
	}, nil
}

func (s *UserServiceImpl) GetUserInfo(targetUserId, curUserId uint64) (*uSrv.UserInfo, error) {
	userModel, err := getUserModelFromCache(int64(targetUserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkg.NewError(pkg.ErrValidation, nil)
		} else {
			return nil, pkg.NewError(pkg.ErrInternal, err)
		}
	}
	info := s.buildUserInfo(*userModel, curUserId)
	return &info, nil
}

func (s *UserServiceImpl) buildUserInfo(targetUser dao.User, curUserId uint64) uSrv.UserInfo {
	var isFollowed bool = false
	if curUserId != 0 {
		isFollowed, _ = s.IsFollowed(int64(targetUser.Id), int64(curUserId))
	}
	followedCnt, _ := s.GetFollowedCnt(int64(targetUser.Id), int64(curUserId))
	followerCnt, _ := s.GetFollowerCnt(int64(targetUser.Id), int64(curUserId))

	return uSrv.UserInfo{
		Id:               targetUser.Id,
		Username:         targetUser.Username,
		Nickname:         targetUser.Nickname,
		AvatarUrl:        targetUser.AvatarUrl,
		BackgroundImgUrl: targetUser.BackgroundImgUrl,
		FollowedCnt:      followedCnt,
		FollowerCnt:      followerCnt,
		IsFollowed:       isFollowed,
	}
}

// todo: Concurrent query
func (s *UserServiceImpl) GetAllFollowed(targetId, userId int64) ([]uSrv.UserInfo, error) {
	models, err := s.GetAllFollowedModels(targetId, userId)
	if err != nil {
		return nil, err
	}

	infos := make([]uSrv.UserInfo, 0, len(models))
	for _, model := range models {
		info := s.buildUserInfo(model, uint64(userId))
		infos = append(infos, info)
	}
	return infos, err
}

// todo: Concurrent query
func (s *UserServiceImpl) GetAllFollower(targetId, userId int64) ([]uSrv.UserInfo, error) {
	models, err := s.GetAllFollowerModels(targetId, userId)
	if err != nil {
		return nil, err
	}

	infos := make([]uSrv.UserInfo, 0, len(models))
	for _, model := range models {
		info := s.buildUserInfo(model, uint64(userId))
		infos = append(infos, info)
	}
	return infos, err
}

func getUserModelFromCache(uid int64) (*dao.User, error) {
	key := fmtUserModelKey(uid)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := setUserModelToCache(uid); err != nil {
			return nil, err
		}
		return getUserModelFromCache(uid)
	}
	model := dao.User{}
	err := cache.Rdb.HGetAll(cache.Ctx, key).Scan(&model)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func setUserModelToCache(uid int64) error {
	key := fmtUserModelKey(uid)
	model := dao.User{}
	if err := dao.Db.Model(&model).Where("id = ?", uid).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pipe := cache.Rdb.Pipeline()
			pipe.HSet(cache.Ctx, key, "null_value", "placeholder")
			pipe.Expire(cache.Ctx, key, cache.NullValTimeout)
			pipe.Exec(cache.Ctx)
		} else {
			return err
		}
	}

	if err := cache.Rdb.HSet(cache.Ctx, key, model).Err(); err != nil {
		return err
	}
	cache.Rdb.Expire(cache.Ctx, key, time.Minute*5+time.Duration(rand.Int32N(5)))
	return nil
}
