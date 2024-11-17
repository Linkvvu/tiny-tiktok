package dao

import "time"

type Video struct {
	Id        int64
	AuthorId  int64
	Title     string
	PlayUrl   string
	CoverUrl  string
	PublishAt time.Time
}

func PersistVideo(video *Video) error {
	return Db.Create(video).Error
}

func GetVideoById(video_id int64) (v Video, err error) {
	err = Db.First(&v, map[string]any{
		"id": video_id,
	}).Error
	return
}

// get published videos by the specified author
func GetVideosByAuthor(author_id int64) ([]Video, error) {
	var videos []Video

	err := Db.Find(
		&videos,
		map[string]any{
			"author_id": author_id,
		},
	).Error

	return videos, err
}
