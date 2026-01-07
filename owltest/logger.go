package owltest

import (
	"context"
	"fmt"
	"sync"
)

// LogEntry captures a log event.
type LogEntry struct {
	Level string
	Msg   string
	Error error
	Args  []any
}

// TestLogger is a mock logger that captures logs in memory.
type TestLogger struct {
	mu      sync.Mutex
	Entries []LogEntry
}

// NewLogger creates a new TestLogger.
func NewLogger() *TestLogger {
	return &TestLogger{}
}

func (l *TestLogger) log(level, msg string, err error, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Entries = append(l.Entries, LogEntry{
		Level: level,
		Msg:   msg,
		Error: err,
		Args:  args,
	})
}

func (l *TestLogger) Debug(ctx context.Context, msg string, args ...any) {
	l.log("DEBUG", msg, nil, args...)
}

func (l *TestLogger) Info(ctx context.Context, msg string, args ...any) {
	l.log("INFO", msg, nil, args...)
}

func (l *TestLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.log("WARN", msg, nil, args...)
}

func (l *TestLogger) Error(ctx context.Context, msg string, err error, args ...any) {
	l.log("ERROR", msg, err, args...)
}

// LastEntry returns the most recent log entry, or nil if empty.
func (l *TestLogger) LastEntry() *LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.Entries) == 0 {
		return nil
	}
	return &l.Entries[len(l.Entries)-1]
}

// Reset clears the log entries.
func (l *TestLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Entries = nil
}

func (l *TestLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fmt.Sprintf("TestLogger{Entries: %d}", len(l.Entries))
}
