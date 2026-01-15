package owl

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCode_String(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{CodeOK, "OK"},
		{CodeInvalid, "INVALID"},
		{CodeUnauthorized, "UNAUTHORIZED"},
		{CodePermissionDenied, "PERMISSION_DENIED"},
		{CodeNotFound, "NOT_FOUND"},
		{CodeInternal, "INTERNAL"},
		{CodeUnavailable, "UNAVAILABLE"},
		{CodeDeadlineExceeded, "DEADLINE_EXCEEDED"},
		{CodeUnknown, "UNKNOWN"},
		{Code(9999), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.code.String(); got != tt.want {
			t.Errorf("Code(%d).String() = %q, want %q", tt.code, got, tt.want)
		}
		if got := tt.code.Error(); got != tt.want {
			t.Errorf("Code(%d).Error() = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestCode_JSON(t *testing.T) {
	// Unmarshal string to code
	tests := []struct {
		input string
		want  Code
	}{
		{`"OK"`, CodeOK},
		{`"INVALID"`, CodeInvalid},
		{`"UNKNOWN"`, CodeUnknown},
		{`"FOOBAR"`, CodeUnknown},
	}
	for _, tt := range tests {
		var c Code
		if err := json.Unmarshal([]byte(tt.input), &c); err != nil {
			t.Errorf("Unmarshal(%s) failed: %v", tt.input, err)
		}
		if c != tt.want {
			t.Errorf("Unmarshal(%s) = %v, want %v", tt.input, c, tt.want)
		}
	}
}

func TestError_Format(t *testing.T) {
	e := &Error{
		Code: CodeInternal,
		Msg:  "boom",
		Op:   "DoThing",
	}
	if e.Error() != "DoThing: boom" {
		t.Errorf("Unexpected format: %s", e.Error())
	}

	e.Err = errors.New("root cause")
	if e.Error() != "DoThing: boom: root cause" {
		t.Errorf("Unexpected format with err: %s", e.Error())
	}
}

func TestProblem_New(t *testing.T) {
	// Test the variadic New helper
	e := New(CodeNotFound, "user not found", errors.New("db error"))
	if e.Code != CodeNotFound {
		t.Error("Wrong code")
	}
	if e.Msg != "user not found" {
		t.Error("Wrong msg")
	}
	if e.Err == nil || e.Err.Error() != "db error" {
		t.Error("Wrong wrapped error")
	}

	// Test functional options via New
	e2 := New(CodeInvalid, WithOp("Validate"))
	if e2.Op != "Validate" {
		t.Error("Functional option failed in New")
	}
}

func TestProblem_WithDetails(t *testing.T) {
	e := Problem(CodeInternal, WithDetails(map[string]any{"k": "v"}))
	if e.Details["k"] != "v" {
		t.Error("Details not set")
	}

	// Append details
	opt := WithDetails(map[string]any{"k2": "v2"})
	opt(e)
	if e.Details["k2"] != "v2" {
		t.Error("Details not appended")
	}
}
