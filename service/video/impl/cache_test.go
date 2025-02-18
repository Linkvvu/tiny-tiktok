package impl

import (
	"log"
	"testing"
)

func TestGetCommentModel(t *testing.T) {
	// err := setCommentModelToCache(dao.Comment{
	// 	Id:          1,
	// 	UserId:      1,
	// 	CommentText: "go test",
	// 	CreateAt:    time.Now(),
	// })
	// if err != nil {
	// 	log.Println(err)
	// 	t.Fatal("failed to set model")
	// }

	c, err := getCommentModelFromCache(1)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(c)
}
