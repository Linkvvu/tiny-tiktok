package service

import (
	"io"
	"sync"
	"tiktok/dao"
	"time"
)

type VideoInfo struct {
	Id         int64    `json:"id"`
	Author     UserInfo `json:"author"`
	PlayUrl    string   `json:"play_url"`
	CoverUrl   string   `json:"cover_url"`
	Title      string   `json:"title"`
	LikeCnt    uint64   `json:"like_count"`
	CommentCnt uint64   `json:"comment_count"`
	IsLike     bool     `json:"is_like"`
	PublishAt  string   `json:"publish_at"`
}

type VideoService interface {
	Publish(user_id int, title string, data io.Reader) error
	List(author_id, user_id int64) ([]VideoInfo, error)
	Recommend(user_id int64, latest_time *time.Time) ([]VideoInfo, error)
	Destroy() error
}

func BuildVideoInfo(like_srv LikeService, video dao.Video, cur_user_id int64) VideoInfo {
	var is_like bool
	var author dao.User
	var like_count uint64

	grp := sync.WaitGroup{}
	grp.Add(3)

	go func() {
		author, _ = dao.GetUserById(video.AuthorId)
		defer grp.Done()
	}()

	go func() {
		is_like, _ = like_srv.HasUserLiked(video.Id, cur_user_id)
		defer grp.Done()
	}()

	go func() {
		like_count, _ = like_srv.LikeCount(video.Id)
		defer grp.Done()
	}()

	grp.Wait()

	return VideoInfo{
		Id:         video.Id,
		Author:     BuildUserInfo(author, cur_user_id),
		PlayUrl:    video.PlayUrl,
		CoverUrl:   video.CoverUrl,
		Title:      video.Title,
		LikeCnt:    like_count,
		CommentCnt: 100000,
		IsLike:     is_like,
		PublishAt:  video.PublishAt.String(),
	}
}
