package alerter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/goware/cachestore"
	"github.com/goware/cachestore/cachestorectl"
	"github.com/goware/cachestore/memlru"
)

type discordAlerter struct {
	Logger        *slog.Logger
	Env           string
	WebhookURL    string
	Username      string
	AvatarURL     string
	MentionRoleID uint64
	SkipLogEntry  bool
	errStore      cachestore.Store[bool]
	Client        *http.Client
}

var _ Alerter = &discordAlerter{}

func NewDiscordAlerter(cfg *Config) (Alerter, error) {
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}
	if cfg.Env == "" {
		return nil, fmt.Errorf("env is required")
	}
	if cfg.Service == "" {
		cfg.Service = "Alerter"
	}

	username, _ := cfg.Extra["username"].(string)
	if username == "" {
		username = cfg.Service
	}

	avatarURL, _ := cfg.Extra["avatarUrl"].(string)
	if avatarURL == "" {
		avatarURL = "https://cdn.discordapp.com/embed/avatars/4.png"
	}

	mentionRoleID, _ := cfg.Extra["mentionRoleId"].(uint64)

	if cfg.AlertCooldown == 0 {
		cfg.AlertCooldown = 1 * time.Minute
	}

	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}
	if cfg.CacheBackend == nil {
		cfg.CacheBackend = memlru.Backend(512)
	}

	mstore, err := cachestorectl.Open[bool](cfg.CacheBackend, cachestore.WithDefaultKeyExpiry(cfg.AlertCooldown))
	if err != nil {
		return nil, err
	}

	return &discordAlerter{
		Logger:        cfg.Logger,
		Env:           cfg.Env,
		WebhookURL:    cfg.WebhookURL,
		Username:      username,
		AvatarURL:     avatarURL,
		MentionRoleID: mentionRoleID,
		SkipLogEntry:  cfg.SkipLogEntry,
		errStore:      mstore,
		Client:        cfg.Client,
	}, nil
}

func (a *discordAlerter) Alert(ctx context.Context, format string, v ...interface{}) {
	// log it
	if !a.SkipLogEntry && a.Logger != nil {
		a.Logger.With("alert", "alert").Error(fmt.Sprintf(format, v...))
	}

	cacheKey := fmt.Sprintf("%d", xxh64FromString(fmt.Sprintf(format, v...)))
	if _, exists, _ := a.errStore.Get(ctx, cacheKey); exists {
		return
	}

	p, err := a.formJsonPayload(format, v...)
	if err != nil {
		if a.Logger != nil {
			a.Logger.With("alert", "alert", "err", err).Error(fmt.Sprintf("failed to form json payload: %v", err))
		}
		return
	}
	a.doRequest(ctx, cacheKey, p)
}

func (a *discordAlerter) doRequest(ctx context.Context, cacheKey string, payload string) {
	req, err := http.NewRequestWithContext(ctx, "POST", a.WebhookURL, bytes.NewReader([]byte(payload)))
	if err != nil {
		if a.Logger != nil {
			a.Logger.With("alert", "alert", "err", err).Error(fmt.Sprintf("failed to create request: %v", err))
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Client.Do(req)
	if err != nil {
		if a.Logger != nil {
			a.Logger.With("alert", "alert", "err", err).Error(fmt.Sprintf("failed to send alert: %v", err))
		}
		return
	}

	defer resp.Body.Close()
	switch statusCode := resp.StatusCode; {
	case (statusCode >= http.StatusOK && statusCode < 300):
		// success - cache the alert
		a.errStore.Set(ctx, cacheKey, true)
		return

	case statusCode == 429:
		if a.Logger != nil {
			a.Logger.With("alert", "alert").Error("alerter has been rate limited")
		}

		timeToWait, err := time.ParseDuration(req.Header.Get("Retry-After"))
		if err != nil {
			if a.Logger != nil {
				a.Logger.With("alert", "alert", "err", err).Error(fmt.Sprintf("failed to parse retry after header: %v", err))
			}
		}

		go func() {
			time.Sleep(timeToWait)
			a.doRequest(ctx, cacheKey, payload)
		}()

	default:
		body, _ := io.ReadAll(resp.Body)
		if a.Logger != nil {
			a.Logger.With("alert", "alert").Error(fmt.Sprintf("unexpected status code: %v, body: %v", resp.StatusCode, string(body)))
		}
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

func (a *discordAlerter) formJsonPayload(format string, v ...interface{}) (string, error) {
	p := payload{
		Username:  a.Username,
		AvatarURL: a.AvatarURL,
		Embeds: []embed{
			{
				Author: struct {
					Name    string `json:"name"`
					IconURL string `json:"icon_url"`
				}{Name: a.Username, IconURL: a.AvatarURL},
				Title:       fmt.Sprintf("Alert - %s", a.Env),
				Description: fmt.Sprintf(format, v...),
				Color:       0xff0000,
			},
		},
	}

	if a.MentionRoleID > 0 {
		p.Content = fmt.Sprintf("<@&%d>", a.MentionRoleID)
	}

	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
