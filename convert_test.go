package owl

import (
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestFromHTTPStatus(t *testing.T) {
	tests := []struct {
		httpStatus int
		want       Code
	}{
		{http.StatusOK, CodeOK},
		{http.StatusCreated, CodeOK},
		{http.StatusBadRequest, CodeInvalid},
		{http.StatusUnauthorized, CodeUnauthorized},
		{http.StatusForbidden, CodePermissionDenied},
		{http.StatusNotFound, CodeNotFound},
		{http.StatusServiceUnavailable, CodeUnavailable},
		{http.StatusGatewayTimeout, CodeDeadlineExceeded},
		{http.StatusInternalServerError, CodeInternal},
		{418, CodeInvalid},  // Generic 4xx
		{502, CodeInternal}, // Generic 5xx
		{999, CodeInternal},
	}

	for _, tt := range tests {
		got := FromHTTPStatus(tt.httpStatus)
		if got != tt.want {
			t.Errorf("FromHTTPStatus(%d) = %v, want %v", tt.httpStatus, got, tt.want)
		}
	}
}

func TestFromGRPCStatus(t *testing.T) {
	tests := []struct {
		grpcCode codes.Code
		want     Code
	}{
		{codes.OK, CodeOK},
		{codes.InvalidArgument, CodeInvalid},
		{codes.Unauthenticated, CodeUnauthorized},
		{codes.PermissionDenied, CodePermissionDenied},
		{codes.NotFound, CodeNotFound},
		{codes.Unavailable, CodeUnavailable},
		{codes.DeadlineExceeded, CodeDeadlineExceeded},
		{codes.Internal, CodeInternal},
		{codes.Unknown, CodeUnknown},
		{codes.DataLoss, CodeUnknown}, // Default fallback
	}

	for _, tt := range tests {
		got := FromGRPCStatus(tt.grpcCode)
		if got != tt.want {
			t.Errorf("FromGRPCStatus(%v) = %v, want %v", tt.grpcCode, got, tt.want)
		}
	}
}
