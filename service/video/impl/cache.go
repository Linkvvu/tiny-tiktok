package impl

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"tiktok/dao"
	"tiktok/middleware/cache"
	"tiktok/pkg"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func fmtVideoModelKey(vid uint64) string {
	return fmt.Sprintf("video_model:%d", vid)
}

func getVideoStreamKey() string {
	return "feed:new"
}

func fmtUserPubVideosKey(uid uint64) string {
	return fmt.Sprintf("user_videos:%d", uid)
}

func fmtUserLikedVideosKey(uid uint64) string {
	return fmt.Sprintf("user_likes:%d", uid)
}

func getLikeMqKey() string {
	return "mq:like"
}

func getVideoModelFromCache(videoId uint64) (dao.Video, error) {
	key := fmtVideoModelKey(videoId)
	values, err := cache.Rdb.HGetAll(cache.Ctx, key).Result()
	if err != nil {
		return dao.Video{}, err
	}
	id, _ := strconv.ParseUint(values["id"], 10, 64)
	authorId, _ := strconv.ParseUint(values["author_id"], 10, 64)
	likeCount, _ := strconv.ParseUint(values["like_count"], 10, 64)
	CommentCount, _ := strconv.ParseUint(values["comment_count"], 10, 64)
	publishAt, _ := strconv.ParseInt(values["publish_at"], 10, 64)

	return dao.Video{
		Id:           id,
		AuthorId:     authorId,
		Title:        values["title"],
		PlayUrl:      values["play_url"],
		CoverUrl:     values["cover_url"],
		LikeCount:    likeCount,
		CommentCount: CommentCount,
		PublishAt:    time.UnixMilli(publishAt),
	}, nil
}

func setVideoModelToCache(v dao.Video) error {
	key := fmtVideoModelKey(v.Id)
	values := map[string]interface{}{
		"id":            v.Id,
		"author_id":     v.AuthorId,
		"title":         v.Title,
		"play_url":      v.PlayUrl,
		"cover_url":     v.CoverUrl,
		"like_count":    v.LikeCount,
		"comment_count": v.CommentCount,
		"publish_at":    v.PublishAt.UnixMilli(),
	}
	return cache.Rdb.HSet(cache.Ctx, key, values).Err()
}

func encodeLikeMqMsg(user_id, video_id uint64, action int8) string {
	return fmt.Sprintf("%d:%d:%d", user_id, video_id, action)
}

func decodeLikeMqMsg(cmd string) (user_id, video_id uint64, action int8) {
	fmt.Sscanf(cmd, "%d:%d:%d", &user_id, &video_id, &action)
	return
}

// todo: use transaction
func likeMqConsumer() {
	sub := cache.Rdb.Subscribe(cache.Ctx, getLikeMqKey())
	defer sub.Close()
	likeChan := sub.Channel()
	for msg := range likeChan {
		cmd := msg.Payload
		uid, vid, act := decodeLikeMqMsg(cmd)
		var incr int
		if act == 0 {
			incr = -1
		} else {
			incr = 1
		}
		err := dao.Db.Model(&dao.Video{}).Where("id = ?", vid).
			Update("like_count", gorm.Expr("like_count + ?", incr)).Error

		if err != nil {
			log.Printf("failed to increment like count for video-%d, skipped, detail: %v\n", vid, err)
		}

		if act == 0 {
			err = dao.DeleteLikeRecord(uid, vid)
			if err != nil {
				log.Printf("failed to delete like record, user-%d video-%d, skipped, detail: %v\n", uid, vid, err)
			}
		} else {
			err = dao.InsertLikeRecord(uid, vid)
			if err != nil {
				log.Printf("failed to insert new record into like table, user-%d video-%d, skipped, detail: %v\n", uid, vid, err)
			}
		}
	}
}

func cacheUserPubVideos(uid uint64) error {
	key := fmtUserPubVideosKey(uid)
	lockKey := fmt.Sprintf("lock:%s", key)
	locked, err := cache.Rdb.SetNX(cache.Ctx, lockKey, 1, 10*time.Second).Result()
	if err != nil {
		err = fmt.Errorf("failed to get lock, key=%s - %w", lockKey, err)
		return pkg.NewError(pkg.ErrInternal, err)
	}

	if !locked {
		return pkg.NewError(pkg.ErrRetry, nil)
	}

	defer cache.Rdb.Del(cache.Ctx, lockKey)

	videoModels, err := dao.GetVideosByAuthor(uid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		err = fmt.Errorf("failed to get videos by user id=%d - %w", uid, err)
		return pkg.NewError(pkg.ErrInternal, err)
	}

	for _, v := range videoModels {
		zData := redis.Z{
			Score:  float64(v.PublishAt.UnixMilli()),
			Member: v.Id,
		}
		err := cache.Rdb.ZAdd(cache.Ctx, key, zData).Err()
		if err != nil {
			log.Printf("WARN: video-%d cache failed, skipped\n", v.Id)
			continue
		}
	}

	// placeholder
	cache.Rdb.ZAdd(cache.Ctx, key, redis.Z{
		Score:  0,
		Member: "",
	})
	cache.Rdb.Expire(cache.Ctx, key, 10*time.Minute)
	return nil
}

func cacheUserLikedVideos(uid uint64) error {
	key := fmtUserLikedVideosKey(uid)
	lockKey := fmt.Sprintf("lock:%s", key)
	locked, err := cache.Rdb.SetNX(cache.Ctx, lockKey, 1, 10*time.Second).Result()
	if err != nil {
		err = fmt.Errorf("failed to get lock, key=%s - %w", lockKey, err)
		return pkg.NewError(pkg.ErrInternal, err)
	}

	if !locked {
		return pkg.NewError(pkg.ErrRetry, nil)
	}

	defer cache.Rdb.Del(cache.Ctx, lockKey)

	vids, err := dao.GetLikedVideoIds(uid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		err = fmt.Errorf("failed to get liked video ids by user id=%d - %w", uid, err)
		return pkg.NewError(pkg.ErrInternal, err)
	}
	for _, v := range vids {
		err := cache.Rdb.SAdd(cache.Ctx, key, interface{}(v)).Err()
		if err != nil {
			log.Printf("WARN: video-%d cache failed, skipped\n", v)
			continue
		}
	}

	// placeholder
	cache.Rdb.SAdd(cache.Ctx, key, "")
	cache.Rdb.Expire(cache.Ctx, key, 10*time.Minute)
	return nil
}

func init() {
	videoModels := []dao.Video{}
	err := dao.Db.Find(&videoModels).Error
	if err != nil {
		log.Fatalln("failed to query video records from DB")
	}

	for _, v := range videoModels {
		err := setVideoModelToCache(v)
		if err != nil {
			log.Printf("WARN: failed to prepare video info for video-%d\n", v.Id)
			continue
		}

		zData := redis.Z{
			Score:  float64(v.PublishAt.UnixMilli()),
			Member: v.Id,
		}
		err = cache.Rdb.ZAdd(cache.Ctx, getVideoStreamKey(), zData).Err()
		if err != nil {
			log.Printf("WARN: failed to add video-%d to feed stream\n", v.Id)
		}
	}

	go likeMqConsumer()
	go commentMqConsumer()
}
