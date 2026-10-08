package http

import (
	"encoding/json"
	"errors"
	apperror "evsched/errors"
	"evsched/http/response"
	"net/http"
)

func WriteJSON(
	writer http.ResponseWriter,
	status int,
	resp any,
) error {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	return json.NewEncoder(writer).Encode(resp)
}

func WriteResponseJSON[T any](
	writer http.ResponseWriter,
	resp response.Response[T],
) error {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(resp.Code)
	return json.NewEncoder(writer).Encode(resp)
}

func WriteSuccess[T any](
	writer http.ResponseWriter,
	data T,
) error {
	resp := response.Success(data)
	return WriteResponseJSON(writer, resp)
}

func WriteSuccessDetail[T any](
	writer http.ResponseWriter,
	status int,
	msg string,
	data T,
) error {
	resp := response.SuccessWithCodeAndMsg(status, msg, data)
	return WriteResponseJSON(writer, resp)
}

func WriteErrorDetail(
	w http.ResponseWriter,
	status int,
	msg string,
) {
	resp := response.Error(status, msg)
	_ = WriteResponseJSON(
		w,
		resp,
	)
}

func WriteError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, apperror.NotFoundErr):
		WriteErrorDetail(w, http.StatusNotFound, err.Error())

	case errors.Is(err, apperror.AlreadyExistsErr):
		WriteErrorDetail(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, apperror.InvalidInputErr):
		WriteErrorDetail(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, apperror.UnauthorizedErr):
		WriteErrorDetail(w, http.StatusUnauthorized, err.Error())

	case errors.Is(err, apperror.ForbiddenErr):
		WriteErrorDetail(w, http.StatusForbidden, err.Error())

	case errors.Is(err, apperror.InternalServerErr):
		WriteErrorDetail(w, http.StatusInternalServerError, err.Error())

	default:
		WriteErrorDetail(
			w,
			http.StatusInternalServerError,
			http.StatusText(http.StatusInternalServerError),
		)
	}
}
