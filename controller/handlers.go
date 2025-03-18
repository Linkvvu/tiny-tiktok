package controller

import (
	"errors"
	"tiktok/pkg"

	"github.com/gin-gonic/gin"
)

func ErrHandler(ctx *gin.Context) {
	ctx.Next()

	errorList := ctx.Errors.ByType(gin.ErrorTypeAny)
	if len(errorList) > 0 {
		// only used the first one
		err := errorList[0].Err

		var appE *pkg.AppError
		if errors.As(err, &appE) {
			// service error
			ctx.JSON(appE.HttpStatus, pkg.NewErrResp(appE))
		} else {
			// native error
			ctx.JSON(appE.HttpStatus, pkg.NewErrResp(appE))
		}

		if gin.Mode() == gin.ReleaseMode {
			ctx.Errors = nil
		}
	}
}
