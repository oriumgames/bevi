package bevi

import (
	"time"

	"github.com/oriumgames/bevi/internal/scheduler"
)

// Diagnostics is the interface for system execution diagnostics.
type Diagnostics interface {
	SystemStart(name string, stage Stage)
	SystemEnd(name string, stage Stage, err error, duration time.Duration)
	EventEmit(name string, count int)
}

// NopDiagnostics is a no-op diagnostics implementation.
type NopDiagnostics struct{}

func (NopDiagnostics) SystemStart(string, Stage)                     {}
func (NopDiagnostics) SystemEnd(string, Stage, error, time.Duration) {}
func (NopDiagnostics) EventEmit(string, int)                         {}

// LogDiagnostics logs diagnostics to a logger interface.
type LogDiagnostics struct {
	log interface{ Printf(string, ...any) }
}

// NewLogDiagnostics creates a diagnostics handler that logs to the given logger.
func NewLogDiagnostics(log interface{ Printf(string, ...any) }) *LogDiagnostics {
	return &LogDiagnostics{log: log}
}

func (d *LogDiagnostics) SystemStart(name string, stage Stage) {
	d.log.Printf("[%s] System %s started", stage, name)
}

func (d *LogDiagnostics) SystemEnd(name string, stage Stage, err error, duration time.Duration) {
	if err != nil {
		d.log.Printf("[%s] System %s finished with error in %v: %v", stage, name, duration, err)
	} else {
		d.log.Printf("[%s] System %s finished in %v", stage, name, duration)
	}
}

func (d *LogDiagnostics) EventEmit(name string, count int) {
	d.log.Printf("Event %s emitted: %d", name, count)
}

// internalDiagnostics adapts bevi.Diagnostics to scheduler.Diagnostics
type internalDiagnostics struct {
	d Diagnostics
}

func (da *internalDiagnostics) SystemStart(name string, stage scheduler.Stage) {
	if da.d != nil {
		da.d.SystemStart(name, Stage(stage))
	}
}

func (da *internalDiagnostics) SystemEnd(name string, stage scheduler.Stage, err error, duration time.Duration) {
	if da.d != nil {
		da.d.SystemEnd(name, Stage(stage), err, duration)
	}
}

func (da *internalDiagnostics) EventEmit(name string, count int) {
	if da.d != nil {
		da.d.EventEmit(name, count)
	}
}
