package owl

import (
	"context"
	"encoding/json"
	"fmt"
)

// Code represents the canonical error code taxonomy.
type Code uint32

const (
	CodeUnknown          Code = 0
	CodeOK               Code = 200
	CodeInvalid          Code = 400 // Invalid Argument
	CodeUnauthorized     Code = 401 // Unauthenticated
	CodePermissionDenied Code = 403 // Permission Denied
	CodeNotFound         Code = 404 // Not Found
	CodeInternal         Code = 500 // Internal System Error
	CodeUnavailable      Code = 503 // Service Unavailable
	CodeDeadlineExceeded Code = 504 // Timeout
)

// Aliases for cleaner API usage (owl.NotFound vs owl.CodeNotFound)
// This matches the user request: owl.Problem(owl.NotFound, ...)
const (
	OK               = CodeOK
	Invalid          = CodeInvalid
	Unauthorized     = CodeUnauthorized
	PermissionDenied = CodePermissionDenied
	NotFound         = CodeNotFound
	Internal         = CodeInternal
	Unavailable      = CodeUnavailable
	DeadlineExceeded = CodeDeadlineExceeded
)

func (c Code) String() string {
	switch c {
	case CodeOK:
		return "OK"
	case CodeInvalid:
		return "INVALID"
	case CodeUnauthorized:
		return "UNAUTHORIZED"
	case CodePermissionDenied:
		return "PERMISSION_DENIED"
	case CodeNotFound:
		return "NOT_FOUND"
	case CodeInternal:
		return "INTERNAL"
	case CodeUnavailable:
		return "UNAVAILABLE"
	case CodeDeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	default:
		return "UNKNOWN"
	}
}

// Error satisfies the stdlib error interface.
func (c Code) Error() string {
	return c.String()
}

// UnmarshalJSON implements custom unmarshaling for Code.
func (c *Code) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "OK":
		*c = CodeOK
	case "INVALID":
		*c = CodeInvalid
	case "UNAUTHORIZED":
		*c = CodeUnauthorized
	case "PERMISSION_DENIED":
		*c = CodePermissionDenied
	case "NOT_FOUND":
		*c = CodeNotFound
	case "INTERNAL":
		*c = CodeInternal
	case "UNAVAILABLE":
		*c = CodeUnavailable
	case "DEADLINE_EXCEEDED":
		*c = CodeDeadlineExceeded
	default:
		*c = CodeUnknown
	}
	return nil
}

// Error is the smart error struct.
type Error struct {
	Code    Code           `json:"code"`
	Msg     string         `json:"message,omitempty"`      // Internal
	SafeMsg string         `json:"safe_message,omitempty"` // Public
	Op      string         `json:"op,omitempty"`
	Err     error          `json:"-"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Msg)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements errors.Is support.
// It checks if the target is an owl.Code and matches,
// OR calls the default Is logic for the wrapped error.
func (e *Error) Is(target error) bool {
	// Check if target is a Code
	if c, ok := target.(Code); ok {
		return e.Code == c
	}
	// Also check pointers to Code if someone passed &CodeNotFound (unlikely but safe)
	if c, ok := target.(*Code); ok {
		return e.Code == *c
	}
	return false
}

// MarshalJSON for RFC 7807 compatibility
func (e *Error) MarshalJSON() ([]byte, error) {
	safeMsg := e.SafeMsg
	if safeMsg == "" {
		safeMsg = e.Code.String()
	}
	return json.Marshal(&struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	}{
		Code:    e.Code.String(),
		Message: safeMsg,
		Details: e.Details,
	})
}

// Logger interface
type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, err error, args ...any)
}

// Monitor interface
type Monitor interface {
	Counter(name string, opts ...MetricOption) Counter
	Histogram(name string, opts ...MetricOption) Histogram
}

type MetricOption func(any)

// Attribute represents a metric tag/label
type Attribute struct {
	Key   string
	Value string
}

// Attr creates a new Attribute.
func Attr(k, v string) Attribute {
	return Attribute{Key: k, Value: v}
}

type Counter interface {
	Inc(ctx context.Context, attrs ...Attribute)
	Add(ctx context.Context, delta float64, attrs ...Attribute)
}

type Histogram interface {
	Record(ctx context.Context, value float64, attrs ...Attribute)
}
