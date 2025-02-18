package controller

import (
	"net/http"
	uSrv "tiktok/service/user"
	"tiktok/util"

	"github.com/gin-gonic/gin"
)

type AuthResp struct {
	response
	UserId uint64 `json:"user_id,omitempty"`
	Token  string `json:"token,omitempty"`
}

type UserInfoResp struct {
	response
	User uSrv.UserInfo `json:"user_info"`
}

type UserController struct {
	userSrv uSrv.UserService
}

func NewUserController(user_srv uSrv.UserService) *UserController {
	return &UserController{
		userSrv: user_srv,
	}
}

func (ctl *UserController) Destroy() {}

type AuthReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ctl *UserController) Register(ctx *gin.Context) {
	var registerReq AuthReq
	err := ctx.ShouldBindJSON(&registerReq)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, AuthResp{
			response: NewErrResponse(util.ErrInvalidParam),
		})
		return
	}

	// TODO:
	//	use validator lib to validate
	if registerReq.Username == "" || registerReq.Password == "" {
		ctx.JSON(http.StatusBadRequest, AuthResp{
			response: NewErrResponse(util.ErrInvalidParam),
		})
		return
	}

	info, err := ctl.userSrv.Register(registerReq.Username, registerReq.Password)
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
	var loginReq AuthReq
	err := ctx.ShouldBindJSON(&loginReq)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, AuthResp{
			response: NewErrResponse(util.ErrInvalidParam),
		})
		return
	}

	info, err := ctl.userSrv.Login(loginReq.Username, loginReq.Password)
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
	var userId uint64
	var err error
	userId = ctx.GetUint64("user_id")
	// targetIdStr := ctx.Query("user_id")
	// if targetIdStr == "" {
	// 	targetId = userId
	// } else {
	// 	targetId, err = strconv.ParseUint(targetIdStr, 10, 64)
	// }

	// info, err := ctl.userSrv.GetInfo(targetId, userId)
	info, err := ctl.userSrv.GetInfo(userId, userId)
	if err != nil {
		ue := util.ConvertOrLog(err)
		ctx.JSON(http.StatusOK, AuthResp{
			response: NewErrResponse(ue),
		})
		return
	}

	ctx.JSON(http.StatusOK, UserInfoResp{
		response: respOk,
		User:     *info,
	})
}
