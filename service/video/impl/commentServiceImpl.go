package impl

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"tiktok/dao"
	"tiktok/middleware/cache"
	"tiktok/pkg"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CommServiceImpl struct{}

func NewCommService() *CommServiceImpl {
	return &CommServiceImpl{}
}

func fmtVideoCommentSetKey(vid int64) string {
	return fmt.Sprintf("video_comments:%d", vid)
}

func fmtVideoCommentModelKey(cid int64) string {
	return fmt.Sprintf("comment_model:%d", cid)
}

func getCommentModelFromCache(cid int64) (dao.Comment, error) {
	key := fmtVideoCommentModelKey(cid)
	result := dao.Comment{}
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := setCommentModelToCache(cid); err != nil {
			return result, err
		}
		return getCommentModelFromCache(cid)
	}
	err := cache.Rdb.HGetAll(cache.Ctx, key).Scan(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// FIXME: Use distributed lock
func setCommentModelToCache(cid int64) error {
	key := fmtVideoCommentModelKey(cid)
	model := dao.Comment{}
	if err := dao.Db.Model(&model).Where("id = ?", cid).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pipe := cache.Rdb.Pipeline()
			pipe.HSet(cache.Ctx, key, "null_value", "placeholder")
			pipe.Expire(cache.Ctx, key, cache.NullValTimeout)
			pipe.Exec(cache.Ctx)
			return nil
		} else {
			return err
		}
	}

	err := cache.Rdb.HSet(cache.Ctx, key, model).Err()
	if err != nil {
		return err
	}
	cache.Rdb.Expire(cache.Ctx, key, time.Minute*5+time.Duration(rand.Int32N(5)))
	return nil
}

func cacheVideoCommentSet(vid int64) error {
	key := fmtVideoCommentSetKey(vid)
	lockKey := fmt.Sprintf("lock:%s", key)
	locked, err := cache.Rdb.SetNX(cache.Ctx, lockKey, 1, time.Minute*10).Result()
	if err != nil {
		err = fmt.Errorf("failed to get lock, key=%s - %w", lockKey, err)
		return pkg.NewError(pkg.ErrInternal, err)
	}

	if !locked {
		return pkg.NewError(pkg.ErrRetry, nil)
	}

	defer cache.Rdb.Del(cache.Ctx, lockKey)

	models := []dao.Comment{}
	err = dao.Db.Model(&dao.Comment{}).Where("video_id = ?", vid).Find(&models).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		err = fmt.Errorf("failed to query comment records on video, video id=%d - %w", vid, err)
		return pkg.NewError(pkg.ErrInternal, err)
	}

	for _, c := range models {
		zData := redis.Z{
			Score:  float64(c.CreateAt),
			Member: c.Id,
		}
		err := cache.Rdb.ZAdd(cache.Ctx, key, zData).Err()
		if err != nil {
			log.Printf("WARN: failed to cache comment for video-%d, skipped\n", c.VideoId)
			continue
		}
	}

	// placeholder
	cache.Rdb.ZAdd(cache.Ctx, key, redis.Z{})
	cache.Rdb.Expire(cache.Ctx, key, 10*time.Minute)
	return nil
}

func getCommentMqKey() string {
	return "mq:delete_comment"
}

func commentMqConsumer() {
	sub := cache.Rdb.Subscribe(cache.Ctx, getCommentMqKey())
	defer sub.Close()

	likeChan := sub.Channel()
	for msg := range likeChan {
		msg := msg.Payload
		var vid, cid int64
		fmt.Sscanf(msg, "%d:%d", &vid, &cid)

		tx := dao.Db.Begin()
		err := tx.Delete(&dao.Comment{Id: cid}).Error
		if err != nil {
			tx.Rollback()
			log.Printf("failed to delete comment record, comment-%d, skipped, detail: %v\n", cid, err)
		}
		err = dao.Db.Model(&dao.Video{}).Where("id = ?", vid).
			Update("comment_count", gorm.Expr("comment_count + ?", -1)).Error
		if err != nil {
			tx.Rollback()
			log.Printf("failed to decr comment count for video-%d, skipped, detail: %v\n", vid, err)
		}
		tx.Commit()
	}
}

// todo: return nil if cache failed but persist successful
func (s *CommServiceImpl) DoComment(videoId, userId, parentId int64, content string) (*dao.Comment, error) {
	// persists
	model := dao.Comment{
		UserId:      userId,
		VideoId:     videoId,
		ParentId:    parentId,
		CommentText: content,
		CreateAt:    time.Now().Unix(),
	}

	tx := dao.Db.Begin()
	err := tx.Create(&model).Error
	if err != nil {
		tx.Rollback()
		err = fmt.Errorf("failed to persist comment, detail: %w", err)
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	err = tx.Model(&dao.Video{}).Where("id = ?", videoId).
		Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	if err != nil {
		tx.Rollback()
		err = fmt.Errorf("failed to incr comment-count for video-%d, detail: %v", videoId, err)
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}
	tx.Commit()

	// cache
	commentSetKey := fmtVideoCommentSetKey(videoId)
	videoModelKey := fmtVideoModelKey(uint64(videoId))

	updateCache := func() error {
		luaScript := redis.NewScript(`
			local comment_set_key = KEYS[1]
			local video_model_key = KEYS[2]

			local comment_id = ARGV[1]
			local timestamp = ARGV[2]

			if redis.call("EXISTS", comment_set_key) == 0 then
				return redis.error_reply("not exist")
			end
			redis.call("ZADD", comment_set_key, timestamp, comment_id)
			redis.call("HINCRBY", video_model_key, 'comment_count', 1)
			return 0
		`)

		return luaScript.Run(cache.Ctx, cache.Rdb,
			[]string{commentSetKey, videoModelKey},
			model.Id, model.CreateAt,
		).Err()
	}

	err = updateCache()
	if err != nil {
		if err.Error() == "not exist" {
			if err := cacheVideoCommentSet(videoId); err != nil {
				return nil, err
			}

			if err = updateCache(); err != nil {
				err := fmt.Errorf("failed to update comment cache, detail: %w", err)
				return nil, pkg.NewError(pkg.ErrInternal, err)
			}
		} else {
			err := fmt.Errorf("failed to update comment cache, detail: %w", err)
			return nil, pkg.NewError(pkg.ErrInternal, err)
		}
	}
	return &model, nil
}

// todo: 检查该请求合理性
func (s *CommServiceImpl) DeleteComment(videoId, commentId, userId int64) error {
	commentSetKey := fmtVideoCommentSetKey(videoId)
	commentModelKey := fmtVideoCommentModelKey(commentId)
	videoModelKey := fmtVideoModelKey(uint64(videoId))
	mqKey := getCommentMqKey()

	msg := fmt.Sprintf("%d:%d", videoId, commentId)
	pipe := cache.Rdb.TxPipeline()
	pipe.ZRem(cache.Ctx, commentSetKey, commentId)
	pipe.Del(cache.Ctx, commentModelKey)
	pipe.Publish(cache.Ctx, mqKey, msg)
	cmds, err := pipe.Exec(cache.Ctx)
	if err != nil {
		err := fmt.Errorf("failed to exec pipeline to delete comment, detail: %w", err)
		return pkg.NewError(pkg.ErrInternal, err)
	}

	for idx, cmd := range cmds {
		if idx == 2 {
			if err := cmd.Err(); err != nil {
				err := fmt.Errorf("failed to publish msg to the delete comment MQ, detail: %w", err)
				return pkg.NewError(pkg.ErrInternal, err)
			}
		}
	}

	// fixme: use lua-script to ensure atomic
	if cache.Rdb.Exists(cache.Ctx, videoModelKey).Val() == 1 {
		cache.Rdb.HIncrBy(cache.Ctx, videoModelKey, "comment_count", -1)
	}
	return nil
}

func (s *CommServiceImpl) GetCommentsOnVideo(vid int64) ([]dao.Comment, error) {
	key := fmtVideoCommentSetKey(vid)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := cacheVideoCommentSet(vid); err != nil {
			return nil, err
		}
		return s.GetCommentsOnVideo(vid)
	}

	idStr, err := cache.Rdb.ZRevRange(cache.Ctx, key, 0, -1).Result()
	if err != nil {
		err := fmt.Errorf("failed to fetch id set of video comments, detail: %w", err)
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	models := make([]dao.Comment, 0, len(idStr))
	for _, str := range idStr {
		if str == "" {
			continue
		}

		id, _ := strconv.ParseInt(str, 10, 64)
		model, err := getCommentModelFromCache(id)
		if err != nil {
			log.Printf("failed to get model of comment-%d, skipped, detail: %v\n", id, err)
			continue
		}
		models = append(models, model)
	}

	return models, nil
}
