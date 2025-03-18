package pkg

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

func NewOkResp() Response {
	return Response{
		Code: 200,
		Msg:  "ok",
	}
}

func NewErrResp(err *AppError) Response {
	return Response{int(err.Code), err.Message}
}
