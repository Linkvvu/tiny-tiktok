package impl

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"sync"
	"tiktok/dao"
	"tiktok/middleware/cache"
	"tiktok/middleware/oss"
	uSrv "tiktok/service/user"
	vSrv "tiktok/service/video"
	"tiktok/util"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type VideoServiceImpl struct {
	// coverRmq *rabbitmq.WorkQueue
	vSrv.LikeService
	vSrv.CommentService
	UserSrv uSrv.UserService
}

func NewVideoService(userSrv uSrv.UserService, likeSrv vSrv.LikeService, commSrv vSrv.CommentService) *VideoServiceImpl {
	return &VideoServiceImpl{
		// coverRmq: pic_queue,
		LikeService:    likeSrv,
		CommentService: commSrv,
		UserSrv:        userSrv,
	}
}

func (s *VideoServiceImpl) Publish(userId uint64, title string, video, thumbnail io.Reader) error {
	// Uploads to OSS
	uuid := uuid.New()
	if err := s.doUpload(uuid.String(), video, thumbnail); err != nil {
		return err
	}

	// Persists to DB
	videoModel := dao.Video{
		AuthorId:  userId,
		Title:     title,
		PlayUrl:   oss.GetUrl(uuid.String(), oss.TypeVideo),
		CoverUrl:  oss.GetUrl(uuid.String(), oss.TypeCover),
		PublishAt: time.Now(),
	}

	err := dao.PersistVideo(&videoModel)
	if err != nil {
		log.Println(err.Error())
		err = fmt.Errorf("%w, %w", util.ErrInternalService, err)
		return err
	}

	// Updates cache
	updateCache := func() (int, error) {
		loadNewVideo := redis.NewScript(`
		local user_videos_key = KEYS[1]
		local video_model_key = KEYS[2]
		local video_stream_key = KEYS[3]
		
		local vid = ARGV[1]
		local author_id = ARGV[2]
		local title = ARGV[3]
		local play_url = ARGV[4]
		local cover_url = ARGV[5]
		local like_count = ARGV[6]
		local publish_at = ARGV[7]
		
		if redis.call("EXISTS", user_videos_key) == 1 then
			redis.call("HMSET", video_model_key,
				"video_id", vid, 
				"author_id", author_id,
				"title", title,
				"play_url", play_url, 
				"cover_url", cover_url, 
				"like_count", like_count, 
				"publish_at", publish_at
			)
			redis.call("ZADD", user_videos_key, publish_at, vid)
			redis.call("ZADD", video_stream_key, publish_at, vid)
			redis.call("EXPIRE", user_videos_key, 600)
			return 0
		else
			return 1
		end
	`)

		res, err := loadNewVideo.Run(cache.Ctx, cache.Rdb, []string{
			fmtUserPubVideosKey(userId),
			fmtVideoModelKey(videoModel.Id),
			getVideoStreamKey(),
		}, videoModel.Id,
			videoModel.AuthorId,
			videoModel.Title,
			videoModel.PlayUrl,
			videoModel.CoverUrl,
			videoModel.LikeCount,
			videoModel.PublishAt.UnixMilli(),
		).Int()
		if err != nil {
			return 0, fmt.Errorf("failed to run lua-script within redis - %w", err)
		}
		return res, nil
	}

	res, err := updateCache()
	if err != nil {
		log.Println(err)
		return util.ErrInternalService
	}

	if res == 1 { // user_videos key isn't exists
		if err := cacheUserPubVideos(userId); err != nil {
			return err
		}
		res, err := updateCache()
		if err != nil {
			log.Println(err)
			return util.ErrInternalService
		}
		if res == 1 {
			fmt.Printf("ERROR: unexpected case, failed to load new video to cache, detail: %v", err)
			return util.ErrInternalService
		}
	}
	return nil
}

// upload to OSS
func (s *VideoServiceImpl) doUpload(name string, video, thumbnail io.Reader) error {
	var err error
	videoObj := oss.OssObject{
		T:    oss.TypeVideo,
		Name: name,
		Data: video,
	}

	err = oss.StoreObject(videoObj)
	if err != nil {
		log.Printf("failed to upload to OSS, detail: %s", err)
		return util.ErrInternalService
	}

	thumbnailObj := oss.OssObject{
		T:    oss.TypeCover,
		Name: name,
		Data: thumbnail,
	}

	err = oss.StoreObject(thumbnailObj)
	if err != nil {
		log.Printf("failed to upload to OSS, detail: %s", err)
		return util.ErrInternalService
	}
	return nil
}

// func (s *VideoServiceImpl) uploadCover(title string) {
// 	s.coverRmq.Publish([]byte(title))
// }

// todo: XXX
func (s *VideoServiceImpl) buildVideoInfo(videoModel dao.Video, userId uint64) vSrv.VideoInfo {
	var isLiked bool
	var authorInfo *uSrv.UserInfo

	grp := sync.WaitGroup{}
	grp.Add(2)
	go func() {
		defer grp.Done()
		isLiked, _ = s.HasUserLiked(videoModel.Id, userId)
	}()

	go func() {
		defer grp.Done()
		authorInfo, _ = s.UserSrv.GetInfo(videoModel.AuthorId, userId)
	}()
	grp.Wait()

	return vSrv.VideoInfo{
		Id:         videoModel.Id,
		Author:     *authorInfo,
		PlayUrl:    videoModel.PlayUrl,
		CoverUrl:   videoModel.CoverUrl,
		Title:      videoModel.Title,
		LikeCnt:    videoModel.LikeCount,
		CommentCnt: 199232,
		IsLike:     isLiked,
		PublishAt:  strconv.FormatInt(videoModel.PublishAt.UnixMilli(), 10),
	}
}

func (s *VideoServiceImpl) ListUserPubVideos(targetId, userId uint64) ([]vSrv.VideoInfo, error) {
	key := fmtUserPubVideosKey(targetId)
	exist := cache.Rdb.Exists(cache.Ctx, key).Val()
	if exist == 0 {
		if err := cacheUserPubVideos(targetId); err != nil {
			return nil, err
		}
		return s.ListUserPubVideos(targetId, userId)
	}

	videoIds, err := cache.Rdb.ZRevRange(cache.Ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("failed to retrieve user pub video set, detail: %v", err)
		return nil, util.ErrInternalService
	}
	videoModels, err := s.retrieveVideosFromCacheStr(videoIds)
	if err != nil {
		log.Printf("failed to retrieve video info, detail: %v", err)
		return nil, util.ErrInternalService
	}

	videosInfos := make([]vSrv.VideoInfo, 0, len(videoModels))
	for _, v := range videoModels {
		videosInfos = append(videosInfos, s.buildVideoInfo(v, userId))
	}
	return videosInfos, nil
}

func (s *VideoServiceImpl) ListUserLikedVideos(targetId, userId uint64) ([]vSrv.VideoInfo, error) {
	key := fmtUserLikedVideosKey(targetId)
	exist := cache.Rdb.Exists(cache.Ctx, key).Val()
	if exist == 0 {
		if err := cacheUserLikedVideos(targetId); err != nil {
			return nil, err
		}
		return s.ListUserLikedVideos(targetId, userId)
	}

	vids, err := cache.Rdb.SMembers(cache.Ctx, key).Result()
	if err != nil {
		log.Printf("failed to retrieve user liked video set, detail: %v", err)
		return nil, util.ErrInternalService
	}

	videoModels, err := s.retrieveVideosFromCacheStr(vids)
	if err != nil {
		log.Printf("failed to retrieve video info, detail: %v", err)
		return nil, util.ErrInternalService
	}
	videosInfos := make([]vSrv.VideoInfo, 0, len(videoModels))
	for _, v := range videoModels {
		videosInfos = append(videosInfos, s.buildVideoInfo(v, userId))
	}
	return videosInfos, nil
}

func (s *VideoServiceImpl) Feed(userId uint64, latestTime *time.Time) ([]vSrv.VideoInfo, error) {
	key := getVideoStreamKey()
	if latestTime == nil {
		latestTime = new(time.Time)
		*latestTime = time.UnixMilli(0)
	}

	timestamp := latestTime.UnixMilli()
	vidStr, err := cache.Rdb.ZRangeByScore(cache.Ctx, key, &redis.ZRangeBy{
		Min:   fmt.Sprintf("(%d", timestamp),
		Max:   "+inf",
		Count: 5,
	}).Result()
	if err != nil {
		log.Printf("WARN: failed to feed, detail: %v", err)
		return nil, util.ErrInternalService
	}

	videoModels, err := s.retrieveVideosFromCacheStr(vidStr)
	if err != nil {
		fmt.Println(err)
	}

	videosInfos := make([]vSrv.VideoInfo, 0, len(videoModels))
	for _, v := range videoModels {
		videosInfos = append(videosInfos, s.buildVideoInfo(v, userId))
	}
	return videosInfos, nil
}

func (s *VideoServiceImpl) buildCommentInfo(model dao.Comment, userId int64) vSrv.CommentInfo {
	userInfo, _ := s.UserSrv.GetInfo(uint64(model.UserId), uint64(userId))
	return vSrv.CommentInfo{
		Id:        model.Id,
		Commenter: *userInfo,
		ParentId:  model.ParentId,
		Content:   model.CommentText,
		CreateAt:  model.CreateAt,
	}
}

func (s *VideoServiceImpl) ListVideoComments(videoId, userId int64) ([]vSrv.CommentInfo, error) {
	models, err := s.GetCommentsOnVideo(videoId)
	if err != nil {
		return nil, err
	}
	commInfos := make([]vSrv.CommentInfo, 0, len(models))
	for _, model := range models {
		commInfo := s.buildCommentInfo(model, userId)
		commInfos = append(commInfos, commInfo)
	}
	return commInfos, nil
}

func (s *VideoServiceImpl) retrieveVideosFromCacheStr(vidStr []string) ([]dao.Video, error) {
	vids := make([]uint64, 0, len(vidStr))
	for _, str := range vidStr {
		if str == "" {
			continue
		}
		vid, _ := strconv.ParseUint(str, 10, 64)
		vids = append(vids, vid)
	}
	return s.retrieveVideosFromCache(vids)
}

// FIXME: Optimized error handling
func (s *VideoServiceImpl) retrieveVideosFromCache(vids []uint64) ([]dao.Video, error) {
	videos := make([]dao.Video, 0, len(vids))
	for _, vid := range vids {
		model, err := getVideoModelFromCache(vid)
		if err != nil {
			fmt.Printf("WARN: video-%d retrieval failed, skipped, detail: %v\n", vid, err)
			continue
		}
		videos = append(videos, model)
	}
	return videos, nil
}
