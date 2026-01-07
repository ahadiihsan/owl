package owl

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToHTTPStatus returns the HTTP status code for a given error.
func ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var e *Error
	if errors.As(err, &e) {
		switch e.Code {
		case CodeOK:
			return http.StatusOK
		case CodeInvalid:
			return http.StatusBadRequest
		case CodeUnauthorized:
			return http.StatusUnauthorized
		case CodePermissionDenied:
			return http.StatusForbidden
		case CodeNotFound:
			return http.StatusNotFound
		case CodeUnavailable:
			return http.StatusServiceUnavailable
		case CodeDeadlineExceeded:
			return http.StatusGatewayTimeout
		case CodeInternal:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// ToGRPCStatus returns the gRPC status for a given error.
func ToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "OK")
	}

	var e *Error
	if errors.As(err, &e) {
		var code codes.Code
		switch e.Code {
		case CodeOK:
			code = codes.OK
		case CodeInvalid:
			code = codes.InvalidArgument
		case CodeUnauthorized:
			code = codes.Unauthenticated
		case CodePermissionDenied:
			code = codes.PermissionDenied
		case CodeNotFound:
			code = codes.NotFound
		case CodeInternal:
			code = codes.Internal
		case CodeUnavailable:
			code = codes.Unavailable
		case CodeDeadlineExceeded:
			code = codes.DeadlineExceeded
		default:
			code = codes.Unknown
		}

		msg := e.SafeMsg
		if msg == "" {
			msg = e.Code.String()
		}

		return status.New(code, msg)
	}

	return status.New(codes.Unknown, "internal server error")
}

// FromHTTPStatus converts an HTTP status code to an owl.Code.
func FromHTTPStatus(code int) Code {
	switch code {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return CodeOK
	case http.StatusBadRequest:
		return CodeInvalid
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodePermissionDenied
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusServiceUnavailable:
		return CodeUnavailable
	case http.StatusGatewayTimeout:
		return CodeDeadlineExceeded
	case http.StatusInternalServerError:
		return CodeInternal
	default:
		// Map generic 4xx to Invalid, 5xx to Internal
		if code >= 400 && code < 500 {
			return CodeInvalid
		}
		if code >= 500 {
			return CodeInternal
		}
		return CodeUnknown
	}
}

// FromGRPCStatus converts a gRPC status code to an owl.Code.
func FromGRPCStatus(code codes.Code) Code {
	switch code {
	case codes.OK:
		return CodeOK
	case codes.InvalidArgument:
		return CodeInvalid
	case codes.Unauthenticated:
		return CodeUnauthorized
	case codes.PermissionDenied:
		return CodePermissionDenied
	case codes.NotFound:
		return CodeNotFound
	case codes.Unavailable:
		return CodeUnavailable
	case codes.DeadlineExceeded:
		return CodeDeadlineExceeded
	case codes.Internal:
		return CodeInternal
	default:
		return CodeUnknown
	}
}
