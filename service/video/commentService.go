package service

import (
	"tiktok/dao"
	uSrv "tiktok/service/user"
)

type CommentInfo struct {
	Id        int64         `json:"id"`
	Commenter uSrv.UserInfo `json:"commenter"`
	ParentId  int64         `json:"parent_id"`
	Content   string        `json:"content"`
	CreateAt  int64         `json:"create_at"`
}

type CommentService interface {
	DoComment(videoId, userId, parentId int64, content string) (*dao.Comment, error)
	DeleteComment(videoId, commentId, userId int64) error

	// used by VideoService
	GetCommentsOnVideo(vid int64) ([]dao.Comment, error)
}
