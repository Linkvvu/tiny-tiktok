package impl

import (
	"fmt"
	"log"
	"strconv"
	"tiktok/dao"
	"tiktok/middleware/cache"
	"tiktok/pkg"
	"time"

	"github.com/redis/go-redis/v9"
)

type relServiceImpl struct{}

func NewRelService() *relServiceImpl {
	return &relServiceImpl{}
}

func encodeFollowMqMsg(targetId, userId int64, action int8) string {
	return fmt.Sprintf("%d:%d:%d", targetId, userId, action)
}

func decodeFollowMqMsg(msg string) (targetId, userId int64, action int8) {
	fmt.Sscanf(msg, "%d:%d:%d", &targetId, &userId, &action)
	return
}

func FollowMqConsumer() {
	sub := cache.Rdb.Subscribe(cache.Ctx, getFollowMqKey())
	defer sub.Close()
	followChan := sub.Channel()
	for msg := range followChan {
		msg := msg.Payload
		tarId, userId, action := decodeFollowMqMsg(msg)
		switch action {
		case 0:
			dao.DeleteFollowRecord(tarId, userId)
		case 1:
			dao.PersistFollow(tarId, userId)
		default:
			log.Printf("FATAL: unknown action in Follow MQ")
		}
	}
}

// todo: 检查请求合理性
func (s *relServiceImpl) DoFollow(targetId, userId int64) error {
	updateCache := func() error {
		followedKey := fmtUserFollowedSetKey(userId)
		followerKey := fmtUserFollowerSetKey(targetId)
		followMqKey := getFollowMqKey()
		luaScript := redis.NewScript(`
			local followedKey = KEYS[1]
			local followerKey = KEYS[2]
			local followMqKey = KEYS[3]
			local followedId = ARGV[1]
			local followerId = ARGV[2]
			local msg = ARGV[3]

			if redis.call("EXISTS", followedKey) == 1 then
				redis.call("SADD", followedKey, followedId)
			end
			if redis.call("EXISTS", followerKey) == 1 then
				redis.call("SADD", followerKey, followerId)
			end
				return redis.call("PUBLISH", followMqKey, msg)
				
		`)
		return luaScript.Run(cache.Ctx, cache.Rdb,
			[]string{followedKey, followerKey, followMqKey},
			targetId, userId, encodeFollowMqMsg(targetId, userId, 1),
		).Err()
	}

	if err := updateCache(); err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}
	return nil
}

// todo: 检查请求合理性
func (s *relServiceImpl) CancelFollow(targetId, userId int64) error {
	followedKey := fmtUserFollowedSetKey(userId)
	followerKey := fmtUserFollowerSetKey(targetId)
	followMqKey := getFollowMqKey()
	cache.Rdb.SRem(cache.Ctx, followedKey, targetId)
	cache.Rdb.SRem(cache.Ctx, followerKey, userId)
	err := cache.Rdb.Publish(cache.Ctx, followMqKey, encodeFollowMqMsg(targetId, userId, 0)).Err()
	if err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}
	return nil
}

// todo: use distributed lock
func (s *relServiceImpl) CacheFollowedSet(uid int64) error {
	key := fmtUserFollowedSetKey(uid)
	models, err := dao.GetFollowedSet(uid)
	if err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}

	pipe := cache.Rdb.Pipeline()
	for _, m := range models {
		pipe.SAdd(cache.Ctx, key, m.FollowedId)
	}
	_, err = pipe.Exec(cache.Ctx)
	if err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}
	cache.Rdb.SAdd(cache.Ctx, key, "")
	cache.Rdb.Expire(cache.Ctx, key, 10*time.Minute)
	return nil
}

// todo: use distributed lock
func (s *relServiceImpl) CacheFollowerSet(uid int64) error {
	key := fmtUserFollowerSetKey(uid)
	models, err := dao.GetFollowerSet(uid)
	if err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}

	pipe := cache.Rdb.Pipeline()
	for _, m := range models {
		pipe.SAdd(cache.Ctx, key, m.UserId)
	}
	_, err = pipe.Exec(cache.Ctx)
	if err != nil {
		return pkg.NewError(pkg.ErrInternal, err)
	}
	cache.Rdb.SAdd(cache.Ctx, key, "")
	cache.Rdb.Expire(cache.Ctx, key, 10*time.Minute)
	return nil
}

func (s *relServiceImpl) GetAllFollowedModels(targetId, userId int64) ([]dao.User, error) {
	key := fmtUserFollowedSetKey(targetId)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := s.CacheFollowedSet(targetId); err != nil {
			return nil, err
		}
		return s.GetAllFollowedModels(targetId, userId)
	}

	uids, err := cache.Rdb.SMembers(cache.Ctx, key).Result()
	if err != nil {
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	userModels, err := s.retrieveUsersFromCacheStr(uids)
	if err != nil {
		return nil, err
	}
	return userModels, nil
}

func (s *relServiceImpl) GetAllFollowerModels(targetId, userId int64) ([]dao.User, error) {
	key := fmtUserFollowerSetKey(targetId)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := s.CacheFollowerSet(targetId); err != nil {
			return nil, err
		}
		return s.GetAllFollowerModels(targetId, userId)
	}

	uids, err := cache.Rdb.SMembers(cache.Ctx, key).Result()
	if err != nil {
		return nil, pkg.NewError(pkg.ErrInternal, err)
	}

	userModels, err := s.retrieveUsersFromCacheStr(uids)
	if err != nil {
		return nil, err
	}
	return userModels, nil
}

func (s *relServiceImpl) IsFollowed(targetId, userId int64) (bool, error) {
	key := fmtUserFollowedSetKey(userId)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := s.CacheFollowedSet(userId); err != nil {
			return false, err
		}
		return s.IsFollowed(targetId, userId)
	}

	followed, err := cache.Rdb.SIsMember(cache.Ctx, key, interface{}(targetId)).Result()
	if err != nil {
		return false, pkg.NewError(pkg.ErrInternal, err)
	}
	return followed, nil
}

// FIXME: 优化占位符处理
func (s *relServiceImpl) GetFollowerCnt(targetId, userId int64) (uint64, error) {
	key := fmtUserFollowerSetKey(targetId)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := s.CacheFollowerSet(targetId); err != nil {
			return 0, err
		}
		return s.GetFollowerCnt(targetId, userId)
	}
	cnt, err := cache.Rdb.SCard(cache.Ctx, key).Uint64()
	if err != nil {
		return 0, pkg.NewError(pkg.ErrInternal, err)
	}
	return cnt - 1, nil
}

// FIXME: 优化占位符处理
func (s *relServiceImpl) GetFollowedCnt(targetId, userId int64) (uint64, error) {
	key := fmtUserFollowedSetKey(targetId)
	if cache.Rdb.Exists(cache.Ctx, key).Val() == 0 {
		if err := s.CacheFollowedSet(targetId); err != nil {
			return 0, err
		}
		return s.GetFollowedCnt(targetId, userId)
	}
	cnt, err := cache.Rdb.SCard(cache.Ctx, key).Uint64()
	if err != nil {
		return 0, pkg.NewError(pkg.ErrInternal, err)
	}
	return cnt - 1, nil
}

func (s *relServiceImpl) retrieveUsersFromCacheStr(uidStr []string) ([]dao.User, error) {
	uids := make([]int64, 0, len(uidStr))
	for _, str := range uidStr {
		if str == "" {
			continue
		}

		uid, _ := strconv.ParseInt(str, 10, 64)
		uids = append(uids, uid)
	}
	return s.retrieveUsersFromCache(uids)
}

func (s *relServiceImpl) retrieveUsersFromCache(uids []int64) ([]dao.User, error) {
	models := make([]dao.User, 0, len(uids))
	for _, uid := range uids {
		model, err := getUserModelFromCache(uid)
		if err != nil {
			fmt.Printf("WARN: video-%d retrieval failed, skipped, detail: %v\n", uid, err)
			continue
		}
		models = append(models, *model)
	}
	return models, nil
}
