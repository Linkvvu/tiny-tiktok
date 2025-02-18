package controller

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"tiktok/middleware/jwt"
	vSrv "tiktok/service/video"
	"tiktok/util"
	"time"

	"github.com/gin-gonic/gin"
)

type DoCommentReq struct {
	Content string `json:"content"`
}

type VideosResp struct {
	response
	Videos []vSrv.VideoInfo `json:"video_list"`
}

type CommentResp struct {
	response
	Comments []vSrv.CommentInfo `json:"comment_list"`
}

type VideoController struct {
	videoSrv vSrv.VideoService
}

func NewVideoController(videoSrv vSrv.VideoService) *VideoController {
	return &VideoController{
		videoSrv: videoSrv,
	}
}

func (ctl *VideoController) Publish(ctx *gin.Context) {
	userId := ctx.GetUint64("user_id")
	title := ctx.PostForm("title")
	var err error
	var videoFH, thumbnailFH *multipart.FileHeader
	videoFH, err = ctx.FormFile("video")
	if err != nil {
		ctx.JSON(http.StatusOK, NewErrResponse(util.ErrInvalidParam))
		return
	}

	thumbnailFH, err = ctx.FormFile("thumbnail")
	if err != nil {
		ctx.JSON(http.StatusOK, NewErrResponse(util.ErrInvalidParam))
		return
	}

	videoFile, _ := videoFH.Open()
	thumbnailFile, _ := thumbnailFH.Open()
	defer videoFile.Close()
	defer thumbnailFile.Close()

	err = ctl.videoSrv.Publish(userId, title, videoFile, thumbnailFile)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *VideoController) ListUserPubVideos(ctx *gin.Context) {
	author_id, _ := strconv.ParseUint(ctx.Param("user_id"), 10, 64)
	user_id := ctx.GetUint64("user_id")
	video_infos, err := ctl.videoSrv.ListUserPubVideos(author_id, user_id)
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

func (ctl *VideoController) ListUserLikedVideos(ctx *gin.Context) {
	author_id, _ := strconv.ParseUint(ctx.Param("user_id"), 10, 64)
	user_id := ctx.GetUint64("user_id")
	videos, err := ctl.videoSrv.ListUserLikedVideos(author_id, user_id)
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

func (ctl *VideoController) Feed(ctx *gin.Context) {
	// user_id is zero when not logged in state
	var user_id uint64
	token := ctx.Query("token")
	if token != "" {
		claim, err := jwt.ParsingToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, NewErrResponse(util.ErrInvalidJwtStatus))
			return
		}

		id, _ := strconv.ParseUint(claim.UserId, 10, 64)
		user_id = id
	}
	// else the user is tourist

	time_str := ctx.Query("latest_time")
	var latest_time *time.Time
	if time_str != "" {
		timestamp, err := strconv.ParseInt(time_str, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, NewErrResponse(util.ErrInvalidParam))
			return
		}
		latest_time = new(time.Time)
		*latest_time = time.UnixMilli(timestamp)
	}

	video_infos, err := ctl.videoSrv.Feed(user_id, latest_time)
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

func (ctl *VideoController) Like(ctx *gin.Context) {
	user_id := ctx.GetUint64("user_id")
	video_id, _ := strconv.ParseUint(ctx.Param("video_id"), 10, 64)
	err := ctl.videoSrv.DoLike(user_id, video_id)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *VideoController) Unlike(ctx *gin.Context) {
	user_id := ctx.GetUint64("user_id")
	video_id, _ := strconv.ParseUint(ctx.Param("video_id"), 10, 64)
	err := ctl.videoSrv.CancelLike(user_id, video_id)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *VideoController) DoComment(ctx *gin.Context) {
	userId := ctx.GetUint64("user_id")
	videoId, _ := strconv.ParseInt(ctx.Param("video_id"), 10, 64)
	parentId, _ := strconv.ParseInt(ctx.Param("parent_id"), 10, 64)
	req := DoCommentReq{}
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, AuthResp{
			response: NewErrResponse(util.ErrInvalidParam),
		})
		return
	}

	err = ctl.videoSrv.DoComment(videoId, int64(userId), parentId, req.Content)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *VideoController) DeleteComment(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	videoId, _ := strconv.ParseInt(ctx.Param("video_id"), 10, 64)
	commId, _ := strconv.ParseInt(ctx.Param("comment_id"), 10, 64)

	err := ctl.videoSrv.DeleteComment(videoId, commId, userId)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}

	ctx.JSON(http.StatusOK, NewErrResponse(util.ErrOk))
}

func (ctl *VideoController) ListVideoComments(ctx *gin.Context) {
	var userId int64
	token := ctx.Query("token")
	if token != "" {
		claim, err := jwt.ParsingToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, NewErrResponse(util.ErrInvalidJwtStatus))
			return
		}

		id, _ := strconv.ParseInt(claim.UserId, 10, 64)
		userId = id
	}

	video_id, _ := strconv.ParseUint(ctx.Param("video_id"), 10, 64)
	commInfos, err := ctl.videoSrv.ListVideoComments(int64(video_id), userId)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, NewErrResponse(ue))
		return
	}
	ctx.JSON(http.StatusOK, CommentResp{
		response: NewErrResponse(util.ErrOk),
		Comments: commInfos,
	})
}
