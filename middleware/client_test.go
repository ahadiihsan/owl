package middleware

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheckResponse_BodyRestoration(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		isJSON     bool
	}{
		{
			name:       "OK Response",
			statusCode: 200,
			body:       `{"status": "ok"}`,
			isJSON:     true,
		},
		{
			name:       "Error Response JSON",
			statusCode: 400,
			body:       `{"code": "INVALID", "message": "bad request"}`,
			isJSON:     true,
		},
		{
			name:       "Error Response Text",
			statusCode: 500,
			body:       "Critical Failure",
			isJSON:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response
			bodyElem := io.NopCloser(strings.NewReader(tt.body))
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       bodyElem,
				Header:     make(http.Header),
			}
			if tt.isJSON {
				resp.Header.Set("Content-Type", "application/json")
			}

			// Call CheckResponse (which reads some body)
			_ = CheckResponse(resp)

			// Verify Body is readable and matches original
			restored, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read restored body: %v", err)
			}

			if string(restored) != tt.body {
				t.Errorf("Body mismatch.\nGot:  %q\nWant: %q", string(restored), tt.body)
			}
		})
	}
}
