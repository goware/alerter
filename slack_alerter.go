package alerter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goware/cachestore"
	"github.com/goware/cachestore/cachestorectl"
	"github.com/goware/cachestore/memlru"
	"github.com/rs/zerolog/log"
)

type SlackConfig struct {
	// required
	// WebhookURL is the discord webhook url for a channel
	WebhookURL string

	// Env is the environment name that will be added to the title
	Env string

	// optionals
	// Username is the username which will appear in the alert message
	Service string

	// AlertCooldown is the time to wait before sending the same alert again
	AlertCooldown time.Duration

	// Skip log entry on alert. In this case, its expected you will log on your own
	SkipLogEntry bool

	Client       *http.Client
	CacheBackend cachestore.Backend
}

type slackAlerter struct {
	Env          string
	WebhookURL   string
	Service      string
	SkipLogEntry bool
	errStore     cachestore.Store[bool]
	Client       *http.Client
}

var _ Alerter = &slackAlerter{}

func NewSlackAlerter(cfg *SlackConfig) (Alerter, error) {
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}
	if cfg.Env == "" {
		return nil, fmt.Errorf("env is required")
	}

	if cfg.Service == "" {
		return nil, fmt.Errorf("service is required")
	}

	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}

	if cfg.AlertCooldown == 0 {
		cfg.AlertCooldown = 1 * time.Minute
	}

	if cfg.CacheBackend == nil {
		cfg.CacheBackend = memlru.Backend(512)
	}

	mstore, err := cachestorectl.Open[bool](cfg.CacheBackend, cachestore.WithDefaultKeyExpiry(cfg.AlertCooldown))
	if err != nil {
		return nil, err
	}

	return &slackAlerter{
		Env:          cfg.Env,
		WebhookURL:   cfg.WebhookURL,
		Service:      cfg.Service,
		SkipLogEntry: cfg.SkipLogEntry,
		errStore:     mstore,
		Client:       cfg.Client,
	}, nil
}

func (a *slackAlerter) Alert(ctx context.Context, format string, v ...interface{}) {
	// log it
	if !a.SkipLogEntry {
		log.Error().Str("alert", "alert").Msgf(format, v...)
	}

	cacheKey := fmt.Sprintf("%d", xxh64FromString(fmt.Sprintf(format, v...)))
	if _, exists, _ := a.errStore.Get(ctx, cacheKey); exists {
		return
	}

	p, err := a.formJsonPayload(format, v...)
	if err != nil {
		log.Error().Str("alert", "alert").Msgf("failed to form json payload: %v", err)
		return
	}
	a.doRequest(ctx, cacheKey, p)
}

func (a *slackAlerter) doRequest(ctx context.Context, cacheKey string, payload string) {
	req, err := http.NewRequestWithContext(ctx, "POST", a.WebhookURL, bytes.NewReader([]byte(payload)))
	if err != nil {
		log.Error().Str("alert", "alert").Msgf("failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Client.Do(req)
	if err != nil {
		log.Error().Str("alert", "alert").Msgf("failed to send alert: %v", err)
		return
	}

	defer resp.Body.Close()
	switch statusCode := resp.StatusCode; {
	case (statusCode >= http.StatusOK && statusCode < 300):
		// success - cache the alert
		a.errStore.Set(ctx, cacheKey, true)
		return
	case statusCode == 429:
		log.Error().Str("alert", "alert").Msgf("rate limited")
		timeToWait, err := time.ParseDuration(req.Header.Get("Retry-After"))
		if err != nil {
			log.Error().Str("alert", "alert").Msgf("failed to parse retry after header: %v", err)
		}

		go func() {
			time.Sleep(timeToWait)
			a.doRequest(ctx, cacheKey, payload)
		}()
	default:
		body, _ := io.ReadAll(resp.Body)
		log.Error().Str("alert", "alert").Msgf("unexpected status code: %v, body: %v", resp.StatusCode, string(body))
	}
}

func (a *slackAlerter) formJsonPayload(format string, v ...interface{}) (string, error) {
	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]interface{}{
					"type":  "plain_text",
					"text":  fmt.Sprintf("Alert: %s - %s", a.Service, a.Env),
					"emoji": true,
				},
			},
			{
				"type": "divider",
			},
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf(format, v...),
				},
			},
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
