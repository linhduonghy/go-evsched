package response

import "net/http"

type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}

func SuccessWithCodeAndMsg[T any](code int, msg string, data T) Response[T] {
	return Response[T]{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}

func Success[T any](data T) Response[T] {
	return SuccessWithCodeAndMsg(http.StatusOK, "", data)
}

func Error(code int, msg string) Response[any] {
	return Response[any]{
		Code: code,
		Msg:  msg,
	}
}
