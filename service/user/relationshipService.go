package service

import "tiktok/dao"

type RelService interface {
	DoFollow(targetId, userId int64) error
	CancelFollow(targetId, userId int64) error
	GetAllFollowerModels(targetId, userId int64) ([]dao.User, error)
	GetAllFollowedModels(targetId, userId int64) ([]dao.User, error)
	IsFollowed(targetId, userId int64) (bool, error)
	GetFollowerCnt(targetId, userId int64) (uint64, error)
	GetFollowedCnt(targetId, userId int64) (uint64, error)
}
