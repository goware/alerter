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
	"github.com/goware/cachestore/memlru"
)

type DiscordConfig struct {
	// required
	// WebhookURL is the discord webhook url for a channel
	WebhookURL string

	// optionals
	// Username is the username which will appear in the alert message
	Username string
	// AvatarURL is the avatar url which will appear in the icon of the alert message
	AvatarURL string
	// RoleIDToPing is the role id to ping in the alert message (if 0, no role will be pinged)
	RoleIDToPing uint64
	// AlertCooldown is the time to wait before sending the same alert again
	AlertCooldown time.Duration
	Client        *http.Client
}

type discordAlerter struct {
	WebhookURL   string
	Username     string
	AvatarURL    string
	RoleIDToPing uint64
	errStore     cachestore.Store[bool]
	Client       *http.Client
}

var _ Alerter = &discordAlerter{}

func NewDiscordAlerter(cfg *DiscordConfig) (Alerter, error) {
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}
	if cfg.Username == "" {
		cfg.Username = "Alerter"
	}
	if cfg.AvatarURL == "" {
		cfg.AvatarURL = "https://cdn.discordapp.com/embed/avatars/4.png"
	}

	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}

	if cfg.AlertCooldown == 0 {
		cfg.AlertCooldown = 1 * time.Minute
	}

	mstore, err := memlru.New[bool](cachestore.WithDefaultKeyExpiry(cfg.AlertCooldown))
	if err != nil {
		return nil, err
	}

	return &discordAlerter{
		WebhookURL:   cfg.WebhookURL,
		Username:     cfg.Username,
		AvatarURL:    cfg.AvatarURL,
		RoleIDToPing: cfg.RoleIDToPing,
		errStore:     mstore,
		Client:       cfg.Client,
	}, nil
}

func (a *discordAlerter) Alert(ctx context.Context, level Level, format string, v ...interface{}) {
	// log it
	level.ZeroLogEvent().Msgf(format, v...)

	cacheKey := fmt.Sprintf("%d", xxh64FromString(fmt.Sprintf(format, v...)))
	if _, exists, _ := a.errStore.Get(ctx, cacheKey); exists {
		return
	}

	p, err := a.formJsonPayload(format, level, v...)
	if err != nil {
		level.ZeroLogEvent().Msgf("failed to form json payload: %v", err)
		return
	}
	a.doRequest(ctx, level, cacheKey, p)
}

func (a *discordAlerter) doRequest(ctx context.Context, level Level, cacheKey string, payload string) {
	req, err := http.NewRequestWithContext(ctx, "POST", a.WebhookURL, bytes.NewReader([]byte(payload)))
	if err != nil {
		level.ZeroLogEvent().Msgf("failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Client.Do(req)
	if err != nil {
		level.ZeroLogEvent().Msgf("failed to send alert: %v", err)
		return
	}

	defer resp.Body.Close()
	switch statusCode := resp.StatusCode; {
	case (statusCode >= http.StatusOK && statusCode < 300):
		// success - cache the alert
		a.errStore.Set(ctx, cacheKey, true)
		return
	case statusCode == 429:
		level.ZeroLogEvent().Msgf("rate limited")
		timeToWait, err := time.ParseDuration(req.Header.Get("Retry-After"))
		if err != nil {
			level.ZeroLogEvent().Msgf("failed to parse retry after header: %v", err)
		}

		go func() {
			time.Sleep(timeToWait)
			a.doRequest(ctx, level, cacheKey, payload)
		}()
	default:
		body, _ := io.ReadAll(resp.Body)
		level.ZeroLogEvent().Msgf("unexpected status code: %v, body: %v", resp.StatusCode, string(body))
	}
}

type embed struct {
	Author struct {
		Name    string `json:"name"`
		IconURL string `json:"icon_url"`
	}
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

type payload struct {
	Username  string  `json:"username"`
	AvatarURL string  `json:"avatar_url"`
	Content   string  `json:"content"`
	Embeds    []embed `json:"embeds"`
}

func (a *discordAlerter) formJsonPayload(format string, level Level, v ...interface{}) (string, error) {
	p := payload{
		Username:  a.Username,
		AvatarURL: a.AvatarURL,
		Embeds: []embed{
			{
				Author: struct {
					Name    string `json:"name"`
					IconURL string `json:"icon_url"`
				}{Name: a.Username, IconURL: a.AvatarURL},
				Title:       "Alert",
				Description: fmt.Sprintf(format, v...),
				Color:       level.Color(),
			},
		},
	}

	if a.RoleIDToPing > 0 && level == LevelError {
		p.Content = fmt.Sprintf("<@&%d>", a.RoleIDToPing)
	}

	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
