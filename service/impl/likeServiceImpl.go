package impl

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"tiktok/dao"
	"tiktok/service"
	"tiktok/util"

	"gorm.io/gorm"
)

type LikeAction = int8

const (
	LikeAct   LikeAction = 1
	CancelAct LikeAction = 2
)

type LikeServiceImpl struct{}

func NewLikeService() *LikeServiceImpl {
	return &LikeServiceImpl{}
}

func (s *LikeServiceImpl) buildVideoInfo(dest *[]service.VideoInfo, vids []int64, cur_user_id int64) {
	grp := sync.WaitGroup{}
	grp.Add(len(vids))

	helper := func(vid int64) {
		defer grp.Done()
		video, _ := dao.GetVideoById(vid)
		vid_info := service.BuildVideoInfo(s, video, cur_user_id)
		*dest = append(*dest, vid_info)
	}

	for _, vid := range vids {
		go helper(vid)
	}

	grp.Wait()
}

func (s *LikeServiceImpl) List(author_id, user_id int64) ([]service.VideoInfo, error) {
	video_ids, err := dao.ListLikedVideoIds(author_id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %w", util.ErrInternalService, err)
	}
	video_infos := make([]service.VideoInfo, 0, len(video_ids))
	s.buildVideoInfo(&video_infos, video_ids, user_id)
	return video_infos, nil
}

func (s *LikeServiceImpl) DoLike(user_id, video_id int64, action int8) error {
	switch action {
	case LikeAct:
		action = 0
	case CancelAct:
		action = 1
	default:
		return util.ErrInvalidParam
	}

	_, err := dao.GetLikeRecord(user_id, video_id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if action == 0 {
				return dao.PerSistLike(&dao.Like{
					UserId:  user_id,
					VideoId: video_id,
					Cancel:  0,
				})
			}
		} else {
			return fmt.Errorf("%w: %w", util.ErrInternalService, err)
		}
	}

	return dao.UpdateLikeAction(user_id, video_id, action)
}

func (s *LikeServiceImpl) IsFavorite(video_id, user_id int64) (bool, error) {
	record, err := dao.GetLikeRecord(user_id, video_id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		log.Printf("unexpected error when invoke <dao.GetLikeRecord>, detail: %s", err)
		return false, util.ErrInternalService
	}

	if record.Cancel == 0 {
		return true, nil
	}
	return false, nil
}

// FIXME:
//
//	use dao.method
func (s *LikeServiceImpl) LikeCount(video_id int64) (uint64, error) {
	var cnt int64
	err := dao.Db.Model(dao.Like{}).Where(map[string]any{
		"video_id": video_id,
		"cancel":   0,
	}).Count(&cnt).Error

	if err != nil {
		fmt.Printf("unexpected error when invoke <dao.LikeCnt>, detail: %s", err)
		return 0, util.ErrInternalService
	}
	return uint64(cnt), nil
}
