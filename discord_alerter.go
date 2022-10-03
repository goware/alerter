package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type Config struct {
	WebhookURL   string
	Username     string
	AvatarURL    string
	RoleIDToPing uint64
	Client       *http.Client
}

type discordAlerter struct {
	WebhookURL   string
	Username     string
	AvatarURL    string
	RoleIDToPing uint64
	Client       *http.Client
}

var _ Alerter = &discordAlerter{}

func NewDiscordAlerter(cfg *Config) (Alerter, error) {
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}
	if cfg.Username == "" {
		cfg.Username = "Alerter"
	}
	if cfg.AvatarURL == "" {
		cfg.AvatarURL = "https://cdn.discordapp.com/embed/avatars/0.png"
	}

	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}

	return &discordAlerter{
		WebhookURL:   cfg.WebhookURL,
		Username:     cfg.Username,
		AvatarURL:    cfg.AvatarURL,
		RoleIDToPing: cfg.RoleIDToPing,
		Client:       cfg.Client,
	}, nil
}

func (a *discordAlerter) Alert(format string, v ...interface{}) {
	// log it
	log.Error().Str("alert", "alert").Msgf(format, v...)
	// TODO:
	// so, lets use the timeBuffer,
	// meaning do not send the same message to Discord more then once within the timeBuffer ..
	// but, lets still call the log.Error() method so we know this is happening repeatedly
	p, err := a.formJsonPayload(format, v...)
	if err != nil {
		log.Error().Str("alert", "alert").Msgf("failed to form json payload: %v", err)
		return
	}
	a.doRequest(p)
}

func (a *discordAlerter) doRequest(payload string) {
	req, err := http.NewRequest("POST", a.WebhookURL, bytes.NewReader([]byte(payload)))
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
		return
	case statusCode == 429:
		log.Error().Str("alert", "alert").Msgf("rate limited")
		timeToWait, err := time.ParseDuration(req.Header.Get("Retry-After"))
		if err != nil {
			log.Error().Str("alert", "alert").Msgf("failed to parse retry after header: %v", err)
		}

		go func() {
			time.Sleep(timeToWait)
			a.doRequest(payload)
		}()
	default:
		body, _ := io.ReadAll(resp.Body)
		log.Error().Str("alert", "alert").Msgf("unexpected status code: %v, body: %v", resp.StatusCode, string(body))
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

type paylod struct {
	Username  string  `json:"username"`
	AvatarURL string  `json:"avatar_url"`
	Content   string  `json:"content"`
	Embeds    []embed `json:"embeds"`
}

func (a *discordAlerter) formJsonPayload(format string, v ...interface{}) (string, error) {
	p := paylod{
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
				Color:       0xff0000,
			},
		},
	}

	if a.RoleIDToPing > 0 {
		p.Content = fmt.Sprintf("<@&%d>", a.RoleIDToPing)
	}

	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
