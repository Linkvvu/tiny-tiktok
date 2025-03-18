package impl

import (
	"fmt"
	"tiktok/middleware/cache"
	"tiktok/pkg"

	"github.com/redis/go-redis/v9"
)

type LikeServiceImpl struct{}

func NewLikeService() *LikeServiceImpl {
	return &LikeServiceImpl{}
}

func (s *LikeServiceImpl) handleLikeAction(user_id, video_id uint64, action int8) error {
	userLikedVideosKey := fmtUserLikedVideosKey(user_id)
	videoInfoKey := fmtVideoModelKey(video_id)
	likeMqKey := getLikeMqKey()

	//	确保点赞数缓存存在
	//	使用lua脚本原子的执行 Exists -> Set(当key不存在) -> Incr -> Expire 命令，
	//	防止出现判断exist为true，但Incr时key正好过期，导致INCR隐式初始化为1
	updateCache := func() (int, error) {
		luaScript := redis.NewScript(`
		local liked_videos_key = KEYS[1]
		local video_info_key = KEYS[2]
		local like_mq_key = KEYS[3]

		local action = tonumber(ARGV[1])
		local vid = ARGV[2]
		local uid = ARGV[3]
		local mq_cmd = ARGV[4]

		local exist = redis.call("EXISTS", liked_videos_key) 
			
		if exist == 1 then
			if redis.call("SISMEMBER", liked_videos_key, vid) == action then
				return 2
			else 
				redis.call( (action == 1) and "SADD" or "SREM", liked_videos_key, vid)
				redis.call("HINCRBY", video_info_key, "like_count", (action == 0) and -1 or 1)
				redis.call("PUBLISH", like_mq_key, mq_cmd)
				return 0
			end
		else
			return 1
		end
	`)
		res, err := luaScript.Run(
			cache.Ctx, cache.Rdb,
			[]string{userLikedVideosKey, videoInfoKey, likeMqKey},
			action, video_id, user_id, encodeLikeMqMsg(user_id, video_id, action),
		).Int()

		if err != nil {
			return res, fmt.Errorf("failed to run lua-script within redis - %w", err)
		}
		return res, nil
	}

	res, err := updateCache()
	if err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}

	// 检查用户点赞状态
	if res == 1 {
		if err := cacheUserLikedVideos(user_id); err != nil {
			return err
		}
		res, err := updateCache()
		if err != nil {
			return pkg.NewError(pkg.ErrInternal, err)
		}

		if res == 1 {
			err = fmt.Errorf("unexpected case, failed to load new video to cache, detail: %w", err)
			return pkg.NewError(pkg.ErrInternal, err)
		} else if res == 2 {
			return pkg.NewError(pkg.ErrValidation, nil)
		}
	} else if res == 2 {
		return pkg.NewError(pkg.ErrValidation, nil)
	}

	return nil
}

func (s *LikeServiceImpl) DoLike(user_id, video_id uint64) error {
	return s.handleLikeAction(user_id, video_id, 1)
}

func (s *LikeServiceImpl) CancelLike(user_id, video_id uint64) error {
	return s.handleLikeAction(user_id, video_id, 0)
}

func (s *LikeServiceImpl) LikeCount(video_id uint64) (uint64, error) {
	videoModel, err := getVideoModelFromCache(video_id)
	if err != nil {
		err = fmt.Errorf("failed to get video model from cache, detail: %w", err)
		return 0, pkg.NewError(pkg.ErrInternal, err)
	}
	return videoModel.LikeCount, nil
}

func (s *LikeServiceImpl) HasUserLiked(video_id, user_id uint64) (bool, error) {
	key := fmtUserLikedVideosKey(user_id)
	exist, err := cache.Rdb.Exists(cache.Ctx, key).Result()
	if err != nil {
		err = fmt.Errorf("failed to execute EXISTS within redis, detail: %w", err)
		return false, pkg.NewError(pkg.ErrInternal, err)
	}

	if exist == 0 {
		err := cacheUserLikedVideos(user_id)
		if err != nil {
			return false, err
		}
	}
	liked, err := cache.Rdb.SIsMember(cache.Ctx, key, interface{}(video_id)).Result()
	if err != nil {
		err = fmt.Errorf("failed to execute SIsMember within redis, detail: %v", err)
		return false, pkg.NewError(pkg.ErrInternal, err)
	}
	return liked, nil
}
