package apperror

import "errors"

var (
	NotFoundErr       = errors.New("entity not found")
	AlreadyExistsErr  = errors.New("entity already exists")
	InvalidInputErr   = errors.New("invalid input")
	ConflictErr       = errors.New("conflict")
	UnauthorizedErr   = errors.New("unauthorized")
	ForbiddenErr      = errors.New("forbidden")
	InternalServerErr = errors.New("riverjob server error")
)

type AppError struct {
	Err error
	Msg string
}

func appError(err error, msg string) *AppError {
	return &AppError{
		Err: err,
		Msg: msg,
	}
}

func (e *AppError) Error() string {
	return e.Msg
}

func (e *AppError) Unwrap() error {
	return e.Err
}
