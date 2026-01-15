package owl

import (
	"errors"
)

// Option defines the functional option pattern for errors.
type Option func(*Error)

// Problem creates a new Error with the given options.
// Usage: owl.Problem(owl.NotFound, owl.WithMsg("user not found"), owl.WithOp("User.Get"))
// Or simply: owl.Problem(owl.Internal, "something went wrong") if we support implicit string args (optional but flexible).
// But stricter functional options are preferred for clarity as requested.
func Problem(code Code, opts ...Option) *Error {
	e := &Error{
		Code: code,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithMsg sets the internal debug message.
func WithMsg(msg string) Option {
	return func(e *Error) {
		e.Msg = msg
	}
}

// WithSafeMsg sets the public facing safe message.
func WithSafeMsg(msg string) Option {
	return func(e *Error) {
		e.SafeMsg = msg
	}
}

// WithOp sets the operation name.
func WithOp(op string) Option {
	return func(e *Error) {
		e.Op = op
	}
}

// WithErr wraps an underlying error.
// If an error is already wrapped, it joins them (Go 1.20+ behavior).
func WithErr(err error) Option {
	return func(e *Error) {
		if e.Err != nil {
			e.Err = errors.Join(e.Err, err)
		} else {
			e.Err = err
		}
	}
}

// WithDetails adds contextual details.
func WithDetails(details map[string]any) Option {
	return func(e *Error) {
		if e.Details == nil {
			e.Details = make(map[string]any)
		}
		for k, v := range details {
			e.Details[k] = v
		}
	}
}

// Legacy-like helper to make simple errors easier?
// The user asked specifically for: owl.Problem(owl.NotFound, "not found")
// This implies mixed variadic arguments OR that the second arg is `any` and checks type.
// To support `owl.Problem(code, "msg")` AND `owl.Problem(code, options...)`, we can use `...any`.

// Refined Problem function to support simple string shorthand.
func New(code Code, args ...any) *Error {
	e := &Error{
		Code: code,
	}
	for _, arg := range args {
		switch v := arg.(type) {
		case Option:
			v(e)
		case string:
			if e.Msg == "" {
				e.Msg = v
			} else {
				// If msg is already set, second string is SafeMsg? Or Op?
				// Let's stick to functional options for clarity unless user *strictly* required shorthand.
				// User Example: `owl.Problem(owl.NotFound, "not found")`
				// This implies implicit Message setting.
				e.Msg = v // Overwrite or append? Overwrite seems standard.
			}
		case error:
			e.Err = v
		}
	}
	return e
}
