package controller

import "tiktok/util"

var respOk = NewErrResponse(util.ErrOk)

type response struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func NewErrResponse(err util.Error) response {
	return response{int(err.Code), err.Msg}
}
