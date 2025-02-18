package dao

type Like struct {
	UserId  uint64
	VideoId uint64
}

func GetLikedVideoIds(userId uint64) ([]uint64, error) {
	videoIds := []uint64{}
	err := Db.Model(Like{}).Where("user_id = ?", userId).Pluck("video_id", &videoIds).Error
	return videoIds, err
}

func GetLikeRecordByVid(video_id uint64) ([]Like, error) {
	like_records := make([]Like, 0)
	err := Db.Model(&Like{}).Where("video_id = ?", video_id).Find(&like_records).Error
	return like_records, err
}

func GetLikeCntByVideoId(video_id uint64) (uint64, error) {
	vid := Video{}
	err := Db.Model(&Video{}).Where("id = ?", video_id).Find(&vid).Error
	if err != nil {
		return 0, err
	}
	return vid.LikeCount, err
}

func InsertLikeRecord(user_id, video_id uint64) error {
	return Db.Create(&Like{UserId: user_id, VideoId: video_id}).Error
}

func DeleteLikeRecord(user_id, video_id uint64) error {
	return Db.Delete(&Like{}, &Like{UserId: user_id, VideoId: video_id}).Error
}
