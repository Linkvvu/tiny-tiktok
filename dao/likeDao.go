package dao

type Like struct {
	UserId  int64
	VideoId int64
}

func GetLikeRecord(user_id, video_id int64) (record Like, err error) {
	err = Db.First(&record, map[string]any{
		"user_id":  user_id,
		"video_id": video_id,
	}).Error
	return
}

func UpdateLikeAction(user_id, video_id int64, cancel int8) error {
	return Db.Model(&Like{}).
		Where("user_id = ? AND video_id = ?", user_id, video_id).
		Update("cancel", cancel).
		Error
}

func ListLikedVideoIds(user_id int64) ([]int64, error) {
	user_ids := []int64{}
	err := Db.Model(Like{}).Where(map[string]any{
		"user_id": user_id,
		"cancel":  0,
	}).Pluck("video_id", &user_ids).Error
	return user_ids, err
}

func PerSistLike(like *Like) error {
	return Db.Create(like).Error
}

func ListLikedRecordByVid(video_id int64) ([]Like, error) {
	like_records := make([]Like, 0)
	err := Db.Model(&Like{}).Where("video_id = ?", video_id).Find(&like_records).Error
	return like_records, err
}

func GetLikeCntByVideoId(video_id int64) (uint64, error) {
	vid := Video{}
	err := Db.Model(&Video{}).Where("id = ?", video_id).Find(&vid).Error
	if err != nil {
		return 0, err
	}
	return vid.LikeCount, err
}

func InsertLikeRecord(user_id, video_id int64) error {
	return Db.Create(&Like{UserId: user_id, VideoId: video_id}).Error
}

func DeleteLikeRecord(user_id, video_id int64) error {
	return Db.Delete(&Like{}, &Like{UserId: user_id, VideoId: video_id}).Error
}
