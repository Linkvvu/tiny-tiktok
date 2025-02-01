package service

type LikeService interface {
	HasUserLiked(video_id, user_id int64) (bool, error)
	LikeCount(video_id int64) (uint64, error)

	DoLike(user_id, video_id int64) error
	CancelLike(user_id, video_id int64) error
	List(author_id, user_id int64) ([]VideoInfo, error)
}
