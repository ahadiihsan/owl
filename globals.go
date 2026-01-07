package owl

import "sync/atomic"

var (
	globalLogger  atomic.Value // Stores loggerHolder
	globalMonitor atomic.Value // Stores monitorHolder
)

type loggerHolder struct {
	l Logger
}

type monitorHolder struct {
	m Monitor
}

func init() {
	globalLogger.Store(loggerHolder{l: NoOpLogger{}})
	globalMonitor.Store(monitorHolder{m: NoOpMonitor{}})
}

// SetLogger sets the global logger instance.
func SetLogger(l Logger) {
	if l != nil {
		globalLogger.Store(loggerHolder{l: l})
	}
}

// GetLogger returns the global logger instance.
func GetLogger() Logger {
	return globalLogger.Load().(loggerHolder).l
}

// SetMonitor sets the global monitor instance.
func SetMonitor(m Monitor) {
	if m != nil {
		globalMonitor.Store(monitorHolder{m: m})
	}
}

// GetMonitor returns the global monitor instance.
func GetMonitor() Monitor {
	return globalMonitor.Load().(monitorHolder).m
}
