package service

type LikeService interface {
	IsFavorite(video_id, user_id int64) (bool, error)
	LikeCount(video_id int64) (uint64, error)

	DoLike(user_id, video_id int64, action int8) error
	List(author_id, user_id int64) ([]VideoInfo, error)
}
