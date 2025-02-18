package service

import "tiktok/dao"

type CommentService interface {
	DoComment(videoId, userId, parentId int64, content string) error
	DeleteComment(videoId, commentId, userId int64) error

	// used by VideoService
	GetCommentsOnVideo(vid int64) ([]dao.Comment, error)
}
