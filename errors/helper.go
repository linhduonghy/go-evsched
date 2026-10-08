package apperror

import "fmt"

func NotFound(entity string) *AppError {
	return appError(NotFoundErr, fmt.Sprintf("%s not found", entity))
}

func AlreadyExists(entity string) *AppError {
	return appError(AlreadyExistsErr, fmt.Sprintf("%s already exists", entity))
}

func InvalidInput(msg string) *AppError {
	return appError(InvalidInputErr, msg)
}

func Conflict(msg string) *AppError {
	return appError(ConflictErr, msg)
}

func Internal(msg string) *AppError {
	return appError(InternalServerErr, msg)
}
