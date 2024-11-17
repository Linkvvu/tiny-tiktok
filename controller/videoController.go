package controller

import (
	"log"
	"net/http"
	"strconv"
	"tiktok/middleware/jwt"
	"tiktok/service"
	"tiktok/util"

	"github.com/gin-gonic/gin"
)

type VideosResp struct {
	response
	Videos []service.VideoInfo `json:"video_list"`
}

type VideoController struct {
	videoSrv service.VideoService
}

func NewVideoController(video_ser service.VideoService) *VideoController {
	return &VideoController{
		videoSrv: video_ser,
	}
}

func (ctl *VideoController) Destroy() {
	err := ctl.videoSrv.Destroy()
	if err != nil {
		log.Panic("failed to destroy video controller")
	}
}

func (ctl *VideoController) Publish(ctx *gin.Context) {
	title := ctx.PostForm("title")
	data, err := ctx.FormFile("data")
	if err != nil {
		log.Println("An unexpected error occurred, detail:", err.Error())
		ctx.JSON(http.StatusOK, NewErrResponse(util.ErrInvalidParam))
		return
	}

	file, _ := data.Open()
	defer file.Close()
	err = ctl.videoSrv.Publish(ctx.GetInt("user_id"), title, file)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *VideoController) List(ctx *gin.Context) {
	author_id, _ := strconv.Atoi(ctx.Query("user_id"))
	user_id := ctx.GetInt("user_id")
	video_infos, err := ctl.videoSrv.List(int64(author_id), int64(user_id))
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}
	ctx.JSON(http.StatusOK, VideosResp{
		response: NewErrResponse(util.ErrOk),
		Videos:   video_infos,
	})
}

func (ctl *VideoController) Feed(ctx *gin.Context) {
	// user_id is zero when not logged in state
	var user_id int64
	token := ctx.Query("token")
	if token != "" {
		claim, err := jwt.ParsingToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, NewErrResponse(util.ErrInvalidJwtStatus))
			return
		}

		id, _ := strconv.Atoi(claim.UserId)
		user_id = int64(id)
	}

	video_infos, err := ctl.videoSrv.Recommend(user_id)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}
	ctx.JSON(http.StatusOK, VideosResp{
		response: NewErrResponse(util.ErrOk),
		Videos:   video_infos,
	})
}
