package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	initControllers()
	defer destroyControllers()

	eng := gin.Default()
	setRoutes(eng)
	eng.Run(":8080")
}
