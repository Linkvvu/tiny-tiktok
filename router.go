package main

import (
	"tiktok/controller"
	"tiktok/middleware/jwt"
	uSrvImp "tiktok/service/user/impl"
	vSrvImp "tiktok/service/video/impl"

	"github.com/gin-gonic/gin"
)

var videoCrl *controller.VideoController
var userCtl *controller.UserController

func initControllers() {
	userSrv := uSrvImp.NewUserService()
	likeSrv := vSrvImp.NewLikeService()
	commSrv := vSrvImp.NewCommService()
	videoSrv := vSrvImp.NewVideoService(userSrv, likeSrv, commSrv)

	videoCrl = controller.NewVideoController(videoSrv)
	userCtl = controller.NewUserController(userSrv)
}

func setRoutes(eng *gin.Engine) {
	tiktok_grp := eng.Group("/tiktok")
	videoGrp := tiktok_grp.Group("/videos")
	// no need AuthorizationMiddleware
	videoGrp.GET("/feed", videoCrl.Feed)
	videoGrp.GET("/:video_id/comments", videoCrl.ListVideoComments)

	// need AuthorizationMiddleware
	videoGrp.Use(jwt.AuthorizationMiddleware)
	videoGrp.POST("", videoCrl.Publish)
	videoGrp.POST("/:video_id/like", videoCrl.Like)
	videoGrp.DELETE("/:video_id/like", videoCrl.Unlike)
	videoGrp.POST("/:video_id/comment/:parent_id", videoCrl.DoComment)
	videoGrp.DELETE("/:video_id/comment/:comment_id", videoCrl.DeleteComment)

	userGrp := tiktok_grp.Group("/users")
	userGrp.POST("/register", userCtl.Register)
	userGrp.POST("/login", userCtl.Login)
	userGrp.Use(jwt.AuthorizationMiddleware)
	userGrp.GET("/:user_id/videos", videoCrl.ListUserPubVideos)
	userGrp.GET("/:user_id/likes", videoCrl.ListUserLikedVideos)
	userGrp.GET("/me", userCtl.GetUserInfo)
}
