package service

import (
	"io"
	uSrv "tiktok/service/user"
	"time"
)

type VideoInfo struct {
	Id         uint64        `json:"id"`
	Author     uSrv.UserInfo `json:"author"`
	PlayUrl    string        `json:"play_url"`
	CoverUrl   string        `json:"cover_url"`
	Title      string        `json:"title"`
	LikeCnt    uint64        `json:"like_count"`
	CommentCnt uint64        `json:"comment_count"`
	IsLike     bool          `json:"is_like"`
	PublishAt  string        `json:"publish_at"`
}

type VideoService interface {
	LikeService
	CommentService
	Publish(userId uint64, title string, video, thumbnail io.Reader) error
	ListUserPubVideos(targetId, userId uint64) ([]VideoInfo, error)
	ListUserLikedVideos(targetId, userId uint64) ([]VideoInfo, error)
	ListVideoComments(videoId, userId int64) ([]CommentInfo, error)
	Feed(userId uint64, latestTime *time.Time) ([]VideoInfo, error)
	MakeComment(videoId, userId, parentId int64, content string) (*CommentInfo, error)
}
