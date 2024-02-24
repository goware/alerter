package main

import (
	"context"

	"github.com/goware/alerter"
)

func main() {
	discordAlerter, err := alerter.NewDiscordAlerter(&alerter.DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/0000000000/abcdefghijklmnopqrstuvwxyz",
		// Username:   "Alerter",
		// AvatarURL:    "https://cdn.discordapp.com/embed/avatars/0.png",
		MentionRoleID: 849690281536389230,
	})

	if err != nil {
		panic(err)
	}

	slackAlerter, err := alerter.NewSlackAlerter(&alerter.SlackConfig{
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
