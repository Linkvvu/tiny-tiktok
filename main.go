package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	initControllers()

	gin.SetMode(gin.ReleaseMode)
	eng := gin.Default()
	setRoutes(eng)
	eng.Run(":8080")
}
