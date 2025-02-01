package controller

import (
	"net/http"
	"strconv"
	"tiktok/service"
	"tiktok/service/impl"
	"tiktok/util"

	"github.com/gin-gonic/gin"
)

type AuthResp struct {
	response
	UserId int64  `json:"user_id,omitempty"`
	Token  string `json:"token,omitempty"`
}

type UserInfoResp struct {
	response
	User service.UserInfo `json:"user_info"`
}

type UserController struct {
	userSrv *impl.UserServiceImpl
}

func NewUserController(user_srv *impl.UserServiceImpl) *UserController {
	return &UserController{
		userSrv: user_srv,
	}
}

func (ctl *UserController) Destroy() {}

func (ctl *UserController) Register(ctx *gin.Context) {
	username := ctx.Query("username")
	password := ctx.Query("password")

	// TODO:
	//	use validator lib to validate
	if username == "" || password == "" {
		ctx.JSON(http.StatusBadRequest, AuthResp{
			response: NewErrResponse(util.ErrInvalidParam),
		})
		return
	}

	info, err := ctl.userSrv.Register(username, password)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, AuthResp{
			response: NewErrResponse(ue),
		})
		return
	}
	ctx.JSON(http.StatusOK, AuthResp{
		response: respOk,
		UserId:   info.Id,
		Token:    info.Token,
	})
}

func (ctl *UserController) Login(ctx *gin.Context) {
	username := ctx.Query("username")
	password := ctx.Query("password")

	info, err := ctl.userSrv.Login(username, password)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, AuthResp{
			response: NewErrResponse(ue),
		})
		return
	}
	ctx.JSON(http.StatusOK, AuthResp{
		response: respOk,
		UserId:   info.Id,
		Token:    info.Token,
	})
}
func (ctl *UserController) GetUserInfo(ctx *gin.Context) {
	var userId, authorId int64
	var err error
	userId = ctx.GetInt64("user_id")
	authorIdStr := ctx.Query("user_id")
	if authorIdStr == "" {
		authorId = userId
	} else {
		authorId, err = strconv.ParseInt(authorIdStr, 10, 64)
	}

	var info service.UserInfo
	info, err = ctl.userSrv.GetInfo(authorId, userId)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, AuthResp{
			response: NewErrResponse(ue),
		})
		return
	}
	ctx.JSON(http.StatusOK, UserInfoResp{
		response: respOk,
		User:     info,
	})
}
