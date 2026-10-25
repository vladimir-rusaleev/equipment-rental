package platform

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalid         = errors.New("invalid input")
	ErrMissing         = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrForbidden       = errors.New("forbidden")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnavailable     = errors.New("dependency unavailable")
)

func GRPCError(err error) error {
	if err == nil {
		return nil
	}
	code := codes.Internal
	switch {
	case errors.Is(err, ErrInvalid):
		code = codes.InvalidArgument
	case errors.Is(err, ErrMissing):
		code = codes.NotFound
	case errors.Is(err, ErrConflict):
		code = codes.FailedPrecondition
	case errors.Is(err, ErrForbidden):
		code = codes.PermissionDenied
	case errors.Is(err, ErrUnauthenticated):
		code = codes.Unauthenticated
	case errors.Is(err, ErrUnavailable):
		code = codes.Unavailable
	case errors.Is(err, context.Canceled):
		code = codes.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		code = codes.DeadlineExceeded
	}
	if code == codes.Internal {
		return status.Error(code, "internal error")
	}
	return status.Error(code, err.Error())
}

func DomainError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument:
		return ErrInvalid
	case codes.NotFound:
		return ErrMissing
	case codes.FailedPrecondition, codes.AlreadyExists:
		return ErrConflict
	case codes.PermissionDenied:
		return ErrForbidden
	case codes.Unauthenticated:
		return ErrUnauthenticated
	default:
		return ErrUnavailable
	}
}
