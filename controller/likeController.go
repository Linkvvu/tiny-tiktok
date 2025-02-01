package controller

import (
	"log"
	"net/http"
	"strconv"
	"tiktok/service"
	"tiktok/util"

	"github.com/gin-gonic/gin"
)

type LikeController struct {
	likeSrv service.LikeService
}

func NewLikeController(like_srv service.LikeService) *LikeController {
	return &LikeController{
		likeSrv: like_srv,
	}
}

type LikeAction = int

const (
	CancelAct LikeAction = 0
	LikeAct   LikeAction = 1
)

func (ctl *LikeController) Like(ctx *gin.Context) {
	user_id := ctx.GetInt64("user_id")
	video_id, _ := strconv.ParseInt(ctx.PostForm("video_id"), 10, 64)
	action_type, err := strconv.Atoi(ctx.DefaultPostForm("action", "none"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, NewErrResponse(util.ErrInvalidParam))
		return
	}

	switch action_type {
	case LikeAct:
		err = ctl.likeSrv.DoLike(user_id, video_id)
	case CancelAct:
		err = ctl.likeSrv.CancelLike(user_id, video_id)
	default:
		ctx.JSON(http.StatusBadRequest, NewErrResponse(util.ErrInvalidParam))
		return
	}

	if err != nil {
		log.Println(err)
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *LikeController) List(ctx *gin.Context) {
	author_id, _ := strconv.Atoi(ctx.Query("user_id"))
	user_id := ctx.GetInt("user_id")
	videos, err := ctl.likeSrv.List(int64(author_id), int64(user_id))
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}
	ctx.JSON(http.StatusOK, VideosResp{
		response: NewErrResponse(util.ErrOk),
		Videos:   videos,
	})
}

func (ctl *LikeController) Destroy() {}
