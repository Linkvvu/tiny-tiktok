package pkg

import (
	"fmt"
	"net/http"
)

type ErrType int

type AppError struct {
	HttpStatus int
	Code       ErrType
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

const (
	ErrInternal ErrType = iota + 1001
	ErrValidation
	ErrAuthException
	ErrRetry
)

const (
	ErrUnmatchedPwd ErrType = iota + 2001
	ErrAccountExisted
)

const ()

var errTypeMap = map[ErrType]AppError{
	ErrInternal: {
		HttpStatus: http.StatusInternalServerError,
		Code:       ErrInternal,
		Message:    "内部服务器错误",
	},
	ErrValidation: {
		HttpStatus: http.StatusBadRequest,
		Code:       ErrValidation,
		Message:    "请求参数无效",
	},
	ErrAuthException: {
		HttpStatus: http.StatusUnauthorized,
		Code:       ErrAuthException,
		Message:    "登录状态异常",
	},
	ErrRetry: {
		HttpStatus: http.StatusServiceUnavailable,
		Code:       ErrRetry,
		Message:    "点击过快，请稍后重试",
	},

	ErrUnmatchedPwd: {
		HttpStatus: http.StatusBadRequest,
		Code:       ErrUnmatchedPwd,
		Message:    "用户名或密码不正确",
	},
	ErrAccountExisted: {
		HttpStatus: http.StatusBadRequest,
		Code:       ErrAccountExisted,
		Message:    "用户名已存在",
	},
}

func NewError(errType ErrType, detail error) *AppError {
	appErr, ok := errTypeMap[errType]
	if !ok {
		appErr = errTypeMap[ErrInternal]
	}

	appErr.Err = detail
	return &appErr
}
