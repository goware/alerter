package main

import (
	"context"
	"log/slog"

	"github.com/goware/alerter"
)

func main() {
	discordAlerter, err := alerter.NewDiscordAlerter(&alerter.Config{
		Logger:     slog.Default(),
		WebhookURL: "https://discord.com/api/webhooks/0000000000/abcdefghijklmnopqrstuvwxyz",
		Env:        "dev",
		Extra: map[string]interface{}{
			// "username":       "Alerter",
			// "avatarUrl":     "https://cdn.discordapp.com/embed/avatars/0.png",
			"mentionRoleId": uint64(849690281536389230),
		},
	})

	if err != nil {
		panic(err)
	}

	slackAlerter, err := alerter.NewSlackAlerter(&alerter.Config{
		Logger:     slog.Default(),
		WebhookURL: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXX",
		Env:        "dev",
		Service:    "test-service",
	})

	if err != nil {
		panic(err)
	}

	slackAlerter.Alert(context.Background(), "hello world %v", "error 1 2 3")
	discordAlerter.Alert(context.Background(), "hello world %v", "error 1 2 3")
}
