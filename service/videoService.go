package service

import (
	"io"
	"sync"
	"tiktok/dao"
)

type VideoInfo struct {
	Id          int64    `json:"id"`
	Author      UserInfo `json:"author"`
	PlayUrl     string   `json:"play_url"`
	CoverUrl    string   `json:"cover_url"`
	Title       string   `json:"title"`
	FavoriteCnt uint64   `json:"favorite_count"`
	CommentCnt  uint64   `json:"comment_count"`
	IsFavorite  bool     `json:"is_favorite"`
}

type VideoService interface {
	Publish(user_id int, title string, data io.Reader) error
	List(author_id, user_id int64) ([]VideoInfo, error)
	Recommend(user_id int64) ([]VideoInfo, error)
	Destroy() error
}

func BuildVideoInfo(like_srv LikeService, video dao.Video, cur_user_id int64) VideoInfo {
	var is_favorite bool
	var author dao.User
	var like_count uint64

	grp := sync.WaitGroup{}
	grp.Add(3)

	go func() {
		author, _ = dao.GetUserById(video.AuthorId)
		defer grp.Done()
	}()

	go func() {
		is_favorite, _ = like_srv.IsFavorite(video.Id, cur_user_id)
		defer grp.Done()
	}()

	go func() {
		like_count, _ = like_srv.LikeCount(video.Id)
		defer grp.Done()
	}()

	grp.Wait()

	return VideoInfo{
		Id:          video.Id,
		Author:      BuildUserInfo(author, cur_user_id),
		PlayUrl:     video.PlayUrl,
		CoverUrl:    video.CoverUrl,
		Title:       video.Title,
		FavoriteCnt: like_count,
		CommentCnt:  100000,
		IsFavorite:  is_favorite,
	}
}
