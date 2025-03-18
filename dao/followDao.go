package dao

type Follow struct {
	UserId     int64
	FollowedId int64
}

func GetFollowedSet(uid int64) ([]Follow, error) {
	models := []Follow{}
	err := Db.Where("user_id = ?", uid).Find(&models).Error
	return models, err
}

func GetFollowerSet(uid int64) ([]Follow, error) {
	models := []Follow{}
	err := Db.Where("followed_id = ?", uid).Find(&models).Error
	return models, err
}

func PersistFollow(followedId, userId int64) error {
	return Db.Create(&Follow{UserId: userId, FollowedId: followedId}).Error
}

func DeleteFollowRecord(followedId, userId int64) error {
	return Db.Delete(&Follow{}, &Follow{UserId: userId, FollowedId: followedId}).Error
}
