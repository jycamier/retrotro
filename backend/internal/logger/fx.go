package logger

import (
	"log/slog"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

var Module = fx.Module("logger")

// FxLogger returns an fx.Option that configures fx to use our slog logger
func FxLogger(cfg Config) fx.Option {
	if !cfg.FxLogs {
		return fx.WithLogger(func() fxevent.Logger {
			return fxevent.NopLogger
		})
	}
	return fx.WithLogger(func() fxevent.Logger {
		return &SlogFxLogger{}
	})
}

// SlogFxLogger adapts slog for fx logging
type SlogFxLogger struct{}

func (l *SlogFxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		slog.Debug("fx: OnStart executing",
			"callee", e.FunctionName,
			"caller", e.CallerName,
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			slog.Error("fx: OnStart failed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"error", e.Err,
			)
		} else {
			slog.Debug("fx: OnStart executed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"runtime", e.Runtime,
			)
		}
	case *fxevent.OnStopExecuting:
		slog.Debug("fx: OnStop executing",
			"callee", e.FunctionName,
			"caller", e.CallerName,
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			slog.Error("fx: OnStop failed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"error", e.Err,
			)
		} else {
			slog.Debug("fx: OnStop executed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"runtime", e.Runtime,
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			slog.Error("fx: supply failed",
				"type", e.TypeName,
				"error", e.Err,
			)
		} else {
			slog.Debug("fx: supplied",
				"type", e.TypeName,
			)
		}
	case *fxevent.Provided:
		if e.Err != nil {
			slog.Error("fx: provide failed",
				"constructor", e.ConstructorName,
				"error", e.Err,
			)
		} else {
			slog.Debug("fx: provided",
				"constructor", e.ConstructorName,
				"types", strings.Join(e.OutputTypeNames, ", "),
			)
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			slog.Error("fx: invoke failed",
				"function", e.FunctionName,
				"error", e.Err,
			)
		} else {
			slog.Debug("fx: invoked",
				"function", e.FunctionName,
			)
		}
	case *fxevent.Started:
		if e.Err != nil {
			slog.Error("fx: start failed", "error", e.Err)
		} else {
			slog.Info("fx: application started")
		}
	case *fxevent.Stopped:
		if e.Err != nil {
			slog.Error("fx: stop failed", "error", e.Err)
		} else {
			slog.Info("fx: application stopped")
		}
	case *fxevent.RollingBack:
		slog.Error("fx: rolling back", "error", e.StartErr)
	case *fxevent.RolledBack:
		if e.Err != nil {
			slog.Error("fx: rollback failed", "error", e.Err)
		} else {
			slog.Info("fx: rolled back")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			slog.Error("fx: logger initialization failed", "error", e.Err)
		} else {
			slog.Debug("fx: logger initialized", "constructor", e.ConstructorName)
		}
	}
}
