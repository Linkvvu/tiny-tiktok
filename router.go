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
	relSrv := uSrvImp.NewRelService()
	userSrv := uSrvImp.NewUserService(relSrv)
	likeSrv := vSrvImp.NewLikeService()
	commSrv := vSrvImp.NewCommService()
	videoSrv := vSrvImp.NewVideoService(userSrv, likeSrv, commSrv)

	videoCrl = controller.NewVideoController(videoSrv)
	userCtl = controller.NewUserController(userSrv)
}

func setRoutes(eng *gin.Engine) {
	tiktok_grp := eng.Group("/tiktok", controller.ErrHandler)
	videoGrp := tiktok_grp.Group("/videos")
	// no need AuthorizationMiddleware
	videoGrp.GET("/feed", videoCrl.Feed)
	videoGrp.GET("/:video_id/comments", videoCrl.ListVideoComments)

	// need AuthorizationMiddleware
	videoGrp.Use(jwt.AuthorizationHandler)
	videoGrp.POST("", videoCrl.Publish)
	videoGrp.POST("/:video_id/like", videoCrl.Like)
	videoGrp.DELETE("/:video_id/like", videoCrl.Unlike)
	videoGrp.POST("/:video_id/comment/:parent_id", videoCrl.DoComment)
	videoGrp.DELETE("/:video_id/comment/:comment_id", videoCrl.DeleteComment)

	userGrp := tiktok_grp.Group("/users")
	// no need AuthorizationMiddleware
	userGrp.POST("/register", userCtl.Register)
	userGrp.POST("/login", userCtl.Login)
	userGrp.GET("/:user_id/videos", videoCrl.ListUserPubVideos)
	userGrp.GET("/:user_id/likes", videoCrl.ListUserLikedVideos)

	// need AuthorizationMiddleware
	userGrp.Use(jwt.AuthorizationHandler)
	userGrp.GET("/me", userCtl.GetUserInfo)
	userGrp.POST(":user_id/follow", userCtl.DoFollow)
	userGrp.DELETE(":user_id/follow", userCtl.CancelFollow)
	userGrp.GET(":user_id/followed", userCtl.GetAllFollowed)
	userGrp.GET(":user_id/follower", userCtl.GetAllFollower)
}
