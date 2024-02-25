package alerter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/goware/cachestore"
)

type Alerter interface {
	Alert(ctx context.Context, format string, v ...interface{})
}

type Config struct {
	// Logger
	Logger *slog.Logger

	// (required) WebhookURL is the discord webhook url for a channel
	WebhookURL string

	// (required) Env is the environment name that will be added to the title
	Env string

	// (required) Service is the name which will appear in the alert message
	Service string

	// (optional) Extra are extra config options to a specific sink
	Extra map[string]interface{}

	// (optional) AlertCooldown is the time to wait before sending the same alert again
	AlertCooldown time.Duration

	// (optional) Skip log entry on alert. In this case, its expected you will log on your own
	SkipLogEntry bool

	// Overrides the default http client and cache backend
	Client       *http.Client
	CacheBackend cachestore.Backend
}

func NewAlerter(destination string, cfg *Config) (Alerter, error) {
	switch destination {
	case "discord":
		return NewDiscordAlerter(cfg)
	case "slack":
		return NewSlackAlerter(cfg)
	default:
		return nil, fmt.Errorf("alerter: unsupported destination '%s'", destination)
	}
}
