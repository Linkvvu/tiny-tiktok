package impl

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"tiktok/dao"
	"tiktok/middleware/cache"
	"tiktok/service"
	"tiktok/util"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type LikeServiceImpl struct{}

func NewLikeService() *LikeServiceImpl {
	// called <syncCacheData> once each new like-service
	go syncCacheData()
	return &LikeServiceImpl{}
}

func encodeMqCmd(user_id, video_id int64, action int8) string {
	return fmt.Sprintf("%d:%d:%d", user_id, video_id, action)
}

func decodeMqCmd(cmd string) (user_id, video_id int64, action int8) {
	fmt.Sscanf(cmd, "%d:%d:%d", &user_id, &video_id, &action)
	return
}

func syncCacheData() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		// sync like action to db
		for {
			cmds, err := cache.Rdb.RPopCount(cache.Ctx, "queue:video_likes", 100).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					break
				}
				log.Println("occur an unexpected error when sync cache data, detail:", err)
				continue
			}

			for _, cmd := range cmds {
				uid, vid, action := decodeMqCmd(cmd)
				if action == 1 {
					if err := dao.InsertLikeRecord(uid, vid); err != nil {
						fmt.Println("occur an unexpected error when call <InsertLikeRecord>, detail:", err)
					}
				} else {
					if err := dao.DeleteLikeRecord(uid, vid); err != nil {
						fmt.Println("occur an unexpected error when call <DeleteLikeRecord>, detail:", err)
					}
				}
			}
		}

		// sync like-count for all videos that in cache
		videoIDs, err := cache.Rdb.SMembers(cache.Ctx, "changed_videos").Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			log.Println("获取变更视频列表失败:", err)
			return
		}

		for _, vidStr := range videoIDs {
			vid, err := strconv.ParseInt(vidStr, 10, 64)
			if err != nil {
				log.Printf("非法 video_id: %s, 错误: %v", vidStr, err)
				continue
			}

			key := fmt.Sprintf("video:like_count:%d", vid)
			likeCount, err := cache.Rdb.Get(cache.Ctx, key).Uint64()
			if err != nil {
				log.Printf("获取点赞数失败: video_id=%d, 错误: %v", vid, err)
				continue
			}

			// 更新数据库
			if err := dao.UpdateLikeCount(vid, likeCount); err != nil {
				log.Printf("更新数据库失败: video_id=%d, 错误: %v", vid, err)
				continue
			}

			// 从变更集合中移除已处理的 video_id
			if err := cache.Rdb.SRem(cache.Ctx, "changed_videos", vid).Err(); err != nil {
				log.Printf("移除变更标记失败: video_id=%d, 错误: %v", vid, err)
			}
		}
	}
}

// todo: 如果用户是游客，则取消无意义的查询
func (s *LikeServiceImpl) buildVideoInfo(dest *[]service.VideoInfo, vids []int64, cur_user_id int64) {
	*dest = make([]service.VideoInfo, len(vids))
	grp := sync.WaitGroup{}
	grp.Add(len(vids))

	for i, vid := range vids {
		go func(idx int, videoID int64) {
			defer grp.Done()
			video, _ := dao.GetVideoById(videoID)
			vidInfo := service.BuildVideoInfo(s, video, cur_user_id)
			(*dest)[idx] = vidInfo
		}(i, vid)
	}

	grp.Wait()
}

// todo: use cache
func (s *LikeServiceImpl) List(author_id, user_id int64) ([]service.VideoInfo, error) {
	video_ids, err := dao.ListLikedVideoIds(author_id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %w", util.ErrInternalService, err)
	}
	video_infos := make([]service.VideoInfo, 0, len(video_ids))
	s.buildVideoInfo(&video_infos, video_ids, user_id)
	return video_infos, nil
}

func (s *LikeServiceImpl) handleLikeAction(user_id, video_id int64, action int8) error {
	likedKey := fmt.Sprintf("video:liked_users:%d", video_id)
	countKey := fmt.Sprintf("video:like_count:%d", video_id)
	lockKey := fmt.Sprintf("lock:%s", countKey)

	// 定义操作参数
	var (
		checkLiked    bool   // 需要检查用户是否已点赞
		redisCountCmd string // Redis计数操作命令（INCR/DECR）
		redisSetCmd   string // Redis集合操作命令（SADD/SREM）
		queueAction   int8   // 同步队列动作标识
	)

	// 根据操作类型设置参数
	switch action {
	case 1: // 点赞
		checkLiked = true
		redisCountCmd = "INCR"
		redisSetCmd = "SADD"
		queueAction = 1
	case 0: // 取消点赞
		checkLiked = false
		redisCountCmd = "DECR"
		redisSetCmd = "SREM"
		queueAction = 0
	default:
		return util.ErrInvalidParam
	}

	// 1. 检查用户点赞状态
	isLiked, err := cache.Rdb.SIsMember(cache.Ctx, likedKey, user_id).Result()
	if err != nil {
		return fmt.Errorf("%w: 查询点赞状态失败 - %w", util.ErrInternalService, err)
	}
	if (checkLiked && isLiked) || (!checkLiked && !isLiked) {
		return util.ErrInvalidParam
	}

	// 2. 确保点赞数缓存存在
	//		使用lua脚本原子的执行 Exists -> Set(当key不存在) -> Incr -> Expire 命令，
	//    防止出现判断exist为true，但Incr时key正好过期，导致INCR隐式初始化为1
	luaScript := redis.NewScript(`
		local count_key = KEYS[1]
		local set_key = KEYS[2]
		local count_action = ARGV[1]
		local set_action = ARGV[2]
		local appended_uid = ARGV[3]
		local expire_time = 3600
		if redis.call("EXISTS", count_key) == 0 then
			return redis.error_reply("not exists")
		end
		redis.call(count_action, count_key)
		redis.call("EXPIRE", count_key, expire_time)
		redis.call(set_action, set_key, appended_uid)
		return 1
	`)
	err = luaScript.Run(
		cache.Ctx, cache.Rdb,
		[]string{countKey, likedKey},
		redisCountCmd, redisSetCmd, strconv.FormatInt(user_id, 10),
	).Err()
	if err != nil {
		if err.Error() == "not exists" { // cache expired
			locked, err := cache.Rdb.SetNX(cache.Ctx, lockKey, 1, 10*time.Second).Result()
			if err != nil {
				return fmt.Errorf("%w: 获取分布式锁失败 - %w", util.ErrInternalService, err)
			}

			if locked {
				defer func() {
					if err := cache.Rdb.Del(cache.Ctx, lockKey).Err(); err != nil {
						log.Printf("警告: 释放锁失败 - key=%s, err=%v", lockKey, err)
					}
				}()

				count, err := dao.GetLikeCntByVideoId(video_id)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						count = 0
					} else {
						return fmt.Errorf("%w: 查询数据库点赞数失败 - %w", util.ErrInternalService, err)
					}
				}

				if err := cache.Rdb.Set(cache.Ctx, countKey, count, 1*time.Hour).Err(); err != nil {
					log.Printf("警告: 设置点赞数缓存失败 - key=%s, err=%v", countKey, err)
				}
				return s.handleLikeAction(user_id, video_id, action)
			} else {
				return util.ErrRetry
			}
		} else { // fatal error
			log.Printf("failed to run lua script in redis, detail: %v", err)
			return util.ErrInternalService
		}
	}

	// 5. 记录同步队列
	if err := cache.Rdb.LPush(cache.Ctx, "queue:video_likes", encodeMqCmd(user_id, video_id, queueAction)).Err(); err != nil {
		log.Printf("严重错误: 无法记录操作到队列 - %v", err)
		// 回滚操作
		reverseAction := map[int8]int8{1: 0, 0: 1}[action]
		s.handleLikeAction(user_id, video_id, reverseAction)
		return fmt.Errorf("%w: 系统繁忙，请稍后重试", util.ErrInternalService)
	}

	// 6. 标记同步视频
	if err := cache.Rdb.SAdd(cache.Ctx, "changed_videos", video_id).Err(); err != nil {
		log.Printf("警告: 标记变更视频失败 - video_id=%d, err=%v", video_id, err)
	}

	return nil
}

func (s *LikeServiceImpl) DoLike(user_id, video_id int64) error {
	return s.handleLikeAction(user_id, video_id, 1)
}

func (s *LikeServiceImpl) CancelLike(user_id, video_id int64) error {
	return s.handleLikeAction(user_id, video_id, 0)
}

func (s *LikeServiceImpl) LikeCount(video_id int64) (uint64, error) {
	key := fmt.Sprintf("video:like_count:%d", video_id)
	cnt, err := cache.Rdb.Get(cache.Ctx, key).Uint64()
	if err == nil {
		return cnt, err
	}

	var video dao.Video
	video, err = dao.GetVideoById(video_id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cache.Rdb.Set(cache.Ctx, key, 0, 5*time.Minute)
			return 0, util.ErrInvalidParam
		}
		return 0, fmt.Errorf("%w: %w", util.ErrInternalService, err)
	}

	cnt = video.LikeCount
	if err = cache.Rdb.Set(cache.Ctx, key, cnt, 1*time.Hour).Err(); err != nil {
		log.Printf("failed to set cache, detail: %v", err)
	}
	return cnt, nil
}

func (s *LikeServiceImpl) HasUserLiked(video_id, user_id int64) (bool, error) {
	key := fmt.Sprintf("video:liked_users:%d", video_id)
	exists, err := cache.Rdb.Exists(cache.Ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %w", util.ErrInternalService, err)
	}

	if exists == 0 {
		records, err := dao.ListLikedRecordByVid(video_id)
		if err != nil {
			return false, fmt.Errorf("%w: %w", util.ErrInternalService, err)
		}

		user_ids := make([]interface{}, 0, len(records))
		for _, record := range records {
			user_ids = append(user_ids, record.UserId)
		}

		err = cache.Rdb.SAdd(cache.Ctx, key, user_ids).Err()
		if err != nil {
			return false, fmt.Errorf("%w: %w", util.ErrInternalService, err)
		}
	}

	var isLiked bool
	if isLiked, err = cache.Rdb.SIsMember(cache.Ctx, key, user_id).Result(); err != nil {
		return false, fmt.Errorf("%w: %w", util.ErrInternalService, err)
	}
	return isLiked, nil
}
