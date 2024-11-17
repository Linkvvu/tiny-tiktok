package dao

type Like struct {
	UserId  int64
	VideoId int64
	Cancel  int8
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
