package owl

import "context"

// NoOpLogger is a logger that does nothing.
type NoOpLogger struct{}

func (NoOpLogger) Debug(ctx context.Context, msg string, args ...any)            {}
func (NoOpLogger) Info(ctx context.Context, msg string, args ...any)             {}
func (NoOpLogger) Warn(ctx context.Context, msg string, args ...any)             {}
func (NoOpLogger) Error(ctx context.Context, msg string, err error, args ...any) {}

// NoOpMonitor is a monitor that does nothing.
type NoOpMonitor struct{}

func (NoOpMonitor) Counter(name string, opts ...MetricOption) Counter {
	return NoOpCounter{}
}
func (NoOpMonitor) Histogram(name string, opts ...MetricOption) Histogram {
	return NoOpHistogram{}
}

type NoOpCounter struct{}

func (NoOpCounter) Inc(ctx context.Context, attrs ...Attribute)                {}
func (NoOpCounter) Add(ctx context.Context, delta float64, attrs ...Attribute) {}

type NoOpHistogram struct{}

func (NoOpHistogram) Record(ctx context.Context, value float64, attrs ...Attribute) {}
