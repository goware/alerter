package alerter

import (
	"context"
	"fmt"
	"log/slog"
)

type defaultAlerter struct {
	logger    *slog.Logger
	logAlerts bool
}

// DefaultAlerter useful when an external source is disabled, but you
// want to offer the interface as a noop (logAlerts=false) or log to
// the logger (logAlerts=true)
func NewDefaultAlerter(logger *slog.Logger, logAlerts bool) Alerter {
	return &defaultAlerter{logAlerts: logAlerts}
}

func (a *defaultAlerter) Alert(ctx context.Context, format string, v ...interface{}) {
	if a.logAlerts {
		a.logger.With("alert", "alert").Error(fmt.Sprintf(format, v...))
	}
}
