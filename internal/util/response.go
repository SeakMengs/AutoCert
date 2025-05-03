package util

import (
	"net/http"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Errors  any    `json:"errors,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func BuildResponseSuccess(data any) Response {
	return Response{
		Success: true,
		Message: constant.REQUEST_SUCCESSFUL,
		Data:    data,
	}
}

func ResponseSuccess(ctx *gin.Context, data any) {
	if data == nil {
		data = gin.H{}
	}

	ctx.JSON(http.StatusOK, BuildResponseSuccess(data))
	ctx.Abort()
}

func BuildResponseFailed(message string, err any, data any) Response {
	if message == "" {
		message = constant.REQUEST_UNSUCCESSFUL
	}

	// TODO: improve this with error check
	// Sometimes we define err type any but err type is error
	if _, ok := err.(error); ok {
		err = GenerateErrorMessages(err.(error))
	}

	if err == nil {
		err = gin.H{}
	}

	if data == nil {
		data = gin.H{}
	}

	return Response{
		Success: false,
		Message: message,
		Errors:  err,
		Data:    data,
	}
}

func ResponseFailed(ctx *gin.Context, code int, message string, err any, data any) {
	ctx.JSON(code, BuildResponseFailed(message, err, data))
	ctx.Abort()
}
