package dao

type Comment struct {
	Id          int64  `redis:"id"`
	UserId      int64  `redis:"user_id"`
	VideoId     int64  `redis:"video_id"`
	ParentId    int64  `redis:"parent_id"`
	CommentText string `redis:"content"`
	CreateAt    int64  `redis:"create_at"`
}
