package owltest

import (
	"context"
	"sync"

	"github.com/myuser/owl"
)

// TestMonitor is a mock monitor that captures metrics in memory.
type TestMonitor struct {
	mu       sync.Mutex
	Counters map[string]float64
}

// NewMonitor creates a new TestMonitor.
func NewMonitor() *TestMonitor {
	return &TestMonitor{
		Counters: make(map[string]float64),
	}
}

func (m *TestMonitor) Counter(name string, opts ...owl.MetricOption) owl.Counter {
	return &testCounter{
		name: name,
		m:    m,
	}
}

func (m *TestMonitor) Histogram(name string, opts ...owl.MetricOption) owl.Histogram {
	return &testHistogram{
		name: name,
		m:    m,
	}
}

// GetCounter returns the current value of a counter.
func (m *TestMonitor) GetCounter(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Counters[name]
}

type testCounter struct {
	name string
	m    *TestMonitor
}

func (c *testCounter) Inc(ctx context.Context, attrs ...owl.Attribute) {
	c.Add(ctx, 1, attrs...)
}

func (c *testCounter) Add(ctx context.Context, delta float64, attrs ...owl.Attribute) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	c.m.Counters[c.name] += delta
}

type testHistogram struct {
	name string
	m    *TestMonitor
}

func (h *testHistogram) Record(ctx context.Context, value float64, attrs ...owl.Attribute) {
	// For now, histograms in owltest just act as counters summing values? or ignore?
	// Let's ignore or just log?
	// The user requirement didn't specify Histogram support in owltest explicit API, but interface needs it.
	// We'll leave it no-op or simple store if needed later.
}
