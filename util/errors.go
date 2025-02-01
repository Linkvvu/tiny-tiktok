package util

import (
	"errors"
	"log"
)

type Code int

type Error struct {
	Code Code
	Msg  string
}

func (e Error) Error() string {
	return e.Msg
}

func ConvertOrLog(err error) Error {
	ue := new(Error)
	if !errors.As(err, ue) {
		log.Fatalln("unexpected error:", err.Error())
		return ErrUnknown
	}
	return *ue
}

var (
	ErrOk               = newError(statusOk)
	ErrUnknown          = newError(statusUnknown)
	ErrSignFailed       = newError(statusSignFailed)
	ErrInvalidParam     = newError(statusInvalidParam)
	ErrAccountExisted   = newError(statusAccountExisted)
	ErrRetry            = newError(statusRetry)
	ErrInternalService  = newError(statusInternalService)
	ErrInvalidJwtStatus = newError(statusInvalidJwtStatus)
)

const (
	statusOk               Code = 0
	statusSignFailed       Code = 401
	statusInvalidParam     Code = 402
	statusAccountExisted   Code = 403
	statusInvalidJwtStatus Code = 404
	statusRetry            Code = 405
	statusInternalService  Code = 500
	statusUnknown          Code = 505
)

var codeMsg map[Code]string = map[Code]string{
	statusOk:               "Ok",
	statusSignFailed:       "账号或密码错误，请检查您的账号密码",
	statusInvalidParam:     "invalid parameters",
	statusAccountExisted:   "用户名已存在",
	statusInvalidJwtStatus: "账号状态异常",
	statusRetry:            "操作过快，请稍后再试",
	statusInternalService:  "内部服务出错，请稍后再试",
	statusUnknown:          "出错啦~ 请重试",
}

func getMsgByCode(code Code) string {
	return codeMsg[code]
}

func newError(code Code) Error {
	return Error{Code: code, Msg: getMsgByCode(code)}
}
