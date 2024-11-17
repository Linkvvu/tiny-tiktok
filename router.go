package main

import (
	"tiktok/controller"
	"tiktok/middleware/jwt"
	"tiktok/middleware/rabbitmq"
	"tiktok/service/impl"
	"tiktok/util"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

var rmq_conn *amqp.Connection
var video_crl *controller.VideoController
var user_ctl *controller.UserController
var like_ctl *controller.LikeController

// FIXME:
//
//	use video-controller substituting like-controller
func initControllers() {
	rmq_conn = rabbitmq.NewRmqConnection(util.AmqpUri)

	video_crl = controller.NewVideoController(
		impl.NewVideoService(
			rabbitmq.NewCoverQueue(rmq_conn),
			impl.NewLikeService(),
		),
	)

	user_ctl = controller.NewUserController(
		impl.NewUserService(),
	)

	like_ctl = controller.NewLikeController(
		impl.NewLikeService(),
	)
}

func destroyControllers() {
	like_ctl.Destroy()
	user_ctl.Destroy()
	video_crl.Destroy()
	rmq_conn.Close()
}

func setRoutes(eng *gin.Engine) {
	tiktok_grp := eng.Group("/douyin")
	tiktok_grp.GET("/feed", video_crl.Feed)

	user_grp := tiktok_grp.Group("/user")
	user_grp.GET("/", jwt.AuthorizationMiddleware, user_ctl.GetUserInfo)
	user_grp.POST("/register/", user_ctl.Register)
	user_grp.POST("/login/", user_ctl.Login)

	video_grp := tiktok_grp.Group("/publish")
	video_grp.Use(jwt.AuthorizationMiddleware)
	video_grp.POST("/action/", video_crl.Publish)
	video_grp.GET("/list/", video_crl.List)

	like_grp := tiktok_grp.Group("/favorite")
	like_grp.Use(jwt.AuthorizationMiddleware)
	like_grp.GET("/list/", like_ctl.List)
	like_grp.POST("/action/", like_ctl.Like)
}
