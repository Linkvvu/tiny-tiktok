package impl

import "fmt"

func fmtUserModelKey(uid int64) string {
	return fmt.Sprintf("user_model:%d", uid)
}

func fmtUserFollowedSetKey(uid int64) string {
	return fmt.Sprintf("user_followed:%d", uid)
}

func fmtUserFollowerSetKey(uid int64) string {
	return fmt.Sprintf("user_followers:%d", uid)
}

func getFollowMqKey() string {
	return "mq:follow"
}

func init() {
	go FollowMqConsumer()
}
