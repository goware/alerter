package alerter

import (
	"context"
	"net/http"
)

type Alerter interface {
	Alert(ctx context.Context, format string, v ...interface{})
	Recoverer() func(next http.Handler) http.Handler
}
