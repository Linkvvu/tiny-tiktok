package impl

import (
	"fmt"
	"io"
	"log"
	"tiktok/dao"
	"tiktok/middleware/oss"
	"tiktok/middleware/rabbitmq"
	"tiktok/service"
	"tiktok/util"
	"time"

	"github.com/google/uuid"
)

type VideoServiceImpl struct {
	coverRmq *rabbitmq.WorkQueue
	likeSrv  service.LikeService
}

func NewVideoService(pic_queue *rabbitmq.WorkQueue, like_srv service.LikeService) *VideoServiceImpl {
	return &VideoServiceImpl{
		coverRmq: pic_queue,
		likeSrv:  like_srv,
	}
}

func (s *VideoServiceImpl) Publish(user_id int, title string, data io.Reader) error {
	uuid := uuid.New()
	if err := s.doUpload(uuid.String(), data); err != nil {
		return err
	}

	video_dao := dao.Video{
		AuthorId:  int64(user_id),
		Title:     title,
		PlayUrl:   oss.GetUrl(uuid.String(), oss.TypeVideo),
		CoverUrl:  oss.GetUrl(uuid.String(), oss.TypeCover),
		PublishAt: time.Now(),
	}
	err := dao.PersistVideo(&video_dao)
	if err != nil {
		log.Println(err.Error())
		err = fmt.Errorf("%w, %w", util.ErrInternalService, err)
	}
	return err
}

func (s *VideoServiceImpl) doUpload(name string, data io.Reader) error {
	videoObj := oss.OssObject{
		T:    oss.TypeVideo,
		Name: name,
		Data: data,
	}

	// stores video
	err := oss.StoreObject(videoObj)
	if err != nil {
		log.Printf("failed to upload to OSS, detail: %s", err)
		return util.ErrInternalService
	}
	// async stores cover by pic-mq
	s.uploadCover(name)
	return nil
}

func (s *VideoServiceImpl) uploadCover(title string) {
	s.coverRmq.Publish([]byte(title))
}

func (s *VideoServiceImpl) List(author_id, user_id int64) ([]service.VideoInfo, error) {
	videos, err := dao.GetVideosByAuthor(author_id)
	if err != nil {
		log.Println(err)
		return nil, util.ErrInternalService
	}

	videos_infos := make([]service.VideoInfo, 0, len(videos))
	for _, v := range videos {
		videos_infos = append(videos_infos, service.BuildVideoInfo(s.likeSrv, v, user_id))
	}
	return videos_infos, err
}

func (s *VideoServiceImpl) Recommend(user_id int64, latest_time *time.Time) ([]service.VideoInfo, error) {
	videos := []dao.Video{}
	if latest_time == nil {
		latest_time = new(time.Time)
		*latest_time = time.Now()
	}
	dao.Db.Where("publish_at < ?", *latest_time).
		Order("publish_at DESC").
		Limit(5).Find(&videos)
	// todo: check SQL error
	videos_infos := make([]service.VideoInfo, 0, len(videos))
	for _, v := range videos {
		videos_infos = append(videos_infos, service.BuildVideoInfo(s.likeSrv, v, user_id))
	}
	return videos_infos, nil
}

func (s *VideoServiceImpl) Destroy() error {
	return s.coverRmq.Close()
}
