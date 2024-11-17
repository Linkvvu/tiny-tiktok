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
	User service.UserInfo `json:"user"`
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
	author_id, _ := strconv.Atoi(ctx.Query("user_id"))
	user_id := ctx.GetInt("user_id")
	info, err := ctl.userSrv.GetInfo(int64(author_id), int64(user_id))
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
