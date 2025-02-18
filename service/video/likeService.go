package service

type LikeService interface {
	HasUserLiked(video_id, user_id uint64) (bool, error)
	LikeCount(video_id uint64) (uint64, error)

	DoLike(user_id, video_id uint64) error
	CancelLike(user_id, video_id uint64) error
}
