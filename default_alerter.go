package alerter

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type defaultAlerter struct {
	logAlerts bool
}

// DefaultAlerter useful when an external source is disabled, but you
// want to offer the interface as a noop (logAlerts=false) or log to
// the logger (logAlerts=true)
func NewDefaultAlerter(logAlerts bool) Alerter {
	return &defaultAlerter{logAlerts: logAlerts}
}

func (a *defaultAlerter) Alert(ctx context.Context, format string, v ...interface{}) {
	if a.logAlerts {
		log.Error().Str("alert", "alert").Msgf(format, v...)
	}
}

func (a *defaultAlerter) Recoverer() func(next http.Handler) http.Handler {
	return middleware.Recoverer
}
