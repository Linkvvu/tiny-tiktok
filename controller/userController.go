package controller

import (
	"net/http"
	"strconv"
	"tiktok/pkg"
	uSrv "tiktok/service/user"

	"github.com/gin-gonic/gin"
)

type AuthResp struct {
	pkg.Response
	UserId uint64 `json:"user_id,omitempty"`
	Token  string `json:"token,omitempty"`
}

type UserInfoResp struct {
	pkg.Response
	User uSrv.UserInfo `json:"user_info"`
}

type UserInfoListResp struct {
	pkg.Response
	Users []uSrv.UserInfo `json:"user_list"`
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
		appE := pkg.NewError(pkg.ErrValidation, err)
		ctx.AbortWithError(appE.HttpStatus, appE)
		return
	}

	// TODO:
	//	use validator lib to validate
	if registerReq.Username == "" || registerReq.Password == "" {
		appE := pkg.NewError(pkg.ErrValidation, err)
		ctx.AbortWithError(appE.HttpStatus, appE)
		return
	}

	info, err := ctl.userSrv.Register(registerReq.Username, registerReq.Password)
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}
	ctx.JSON(http.StatusOK, AuthResp{
		Response: pkg.NewOkResp(),
		UserId:   info.Id,
		Token:    info.Token,
	})
}

func (ctl *UserController) Login(ctx *gin.Context) {
	var loginReq AuthReq
	err := ctx.ShouldBindJSON(&loginReq)
	if err != nil {
		appE := pkg.NewError(pkg.ErrValidation, err)
		ctx.AbortWithError(appE.HttpStatus, appE)
		return
	}

	info, err := ctl.userSrv.Login(loginReq.Username, loginReq.Password)
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, AuthResp{
		Response: pkg.NewOkResp(),
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
	info, err := ctl.userSrv.GetUserInfo(userId, userId)
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, UserInfoResp{
		Response: pkg.NewOkResp(),
		User:     *info,
	})
}

func (ctl *UserController) DoFollow(ctx *gin.Context) {
	userId := ctx.GetUint64("user_id")
	targetId, err := strconv.ParseInt(ctx.Param("user_id"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, pkg.NewError(pkg.ErrValidation, err))
		return
	}
	err = ctl.userSrv.DoFollow(targetId, int64(userId))
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, pkg.NewOkResp())
}

func (ctl *UserController) CancelFollow(ctx *gin.Context) {
	userId := ctx.GetUint64("user_id")
	targetId, err := strconv.ParseInt(ctx.Param("user_id"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, pkg.NewError(pkg.ErrValidation, err))
		return
	}
	err = ctl.userSrv.CancelFollow(targetId, int64(userId))
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, pkg.NewOkResp())
}

func (ctl *UserController) GetAllFollowed(ctx *gin.Context) {
	userId := ctx.GetUint64("user_id")
	targetId, err := strconv.ParseInt(ctx.Param("user_id"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, pkg.NewError(pkg.ErrValidation, err))
		return
	}
	userInfos, err := ctl.userSrv.GetAllFollowed(targetId, int64(userId))
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, UserInfoListResp{
		Response: pkg.NewOkResp(),
		Users:    userInfos,
	})
}

func (ctl *UserController) GetAllFollower(ctx *gin.Context) {
	userId := ctx.GetUint64("user_id")
	targetId, err := strconv.ParseInt(ctx.Param("user_id"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, pkg.NewError(pkg.ErrValidation, err))
		return
	}
	userInfos, err := ctl.userSrv.GetAllFollower(targetId, int64(userId))
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, UserInfoListResp{
		Response: pkg.NewOkResp(),
		Users:    userInfos,
	})
}
