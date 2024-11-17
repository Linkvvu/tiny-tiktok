package controller

import (
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

func (ctl *LikeController) Like(ctx *gin.Context) {
	video_id, _ := strconv.Atoi(ctx.Query("video_id"))
	action_type, _ := strconv.Atoi(ctx.Query("action_type"))
	user_id := ctx.GetInt("user_id")
	err := ctl.likeSrv.DoLike(int64(user_id), int64(video_id), int8(action_type))
	if err != nil {
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
