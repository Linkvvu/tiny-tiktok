package dao

import "time"

type Video struct {
	Id        uint64
	AuthorId  uint64
	Title     string
	PlayUrl   string
	CoverUrl  string
	PublishAt time.Time
	LikeCount uint64
}

func PersistVideo(video *Video) error {
	return Db.Create(video).Error
}

func GetVideoById(video_id uint64) (v Video, err error) {
	err = Db.First(&v, map[string]any{
		"id": video_id,
	}).Error
	return
}

// get published videos by the specified author
func GetVideosByAuthor(author_id uint64) ([]Video, error) {
	var videos []Video

	err := Db.Find(
		&videos,
		map[string]any{
			"author_id": author_id,
		},
	).Error

	return videos, err
}

func UpdateLikeCount(video_id, count uint64) error {
	return Db.Model(&Video{}).Where("id = ?", video_id).Update("like_count", count).Error
}

func GetVideosByIds(ids []uint64) []Video {
	videos := make([]Video, 0, len(ids))
	for _, vid := range ids {
		video, _ := GetVideoById(vid)
		videos = append(videos, video)
	}
	return videos
}
