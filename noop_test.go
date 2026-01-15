package owl

import (
	"context"
	"errors"
	"testing"
)

func TestNoOpLogger(t *testing.T) {
	l := NoOpLogger{}
	ctx := context.Background()

	l.Debug(ctx, "msg")
	l.Info(ctx, "msg")
	l.Warn(ctx, "msg")
	l.Error(ctx, "msg", errors.New("err"))
}

func TestNoOpMonitor(t *testing.T) {
	m := NoOpMonitor{}
	ctx := context.Background()

	c := m.Counter("c")
	c.Inc(ctx)
	c.Add(ctx, 1)

	h := m.Histogram("h")
	h.Record(ctx, 1)
}
