package main

import (
	"context"

	"github.com/goware/alerter"
)

func main() {
	alerter, err := alerter.NewDiscordAlerter(&alerter.DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/0000000000/abcdefghijklmnopqrstuvwxyz",
		// Username:   "Alerter",
		// AvatarURL:    "https://cdn.discordapp.com/embed/avatars/0.png",
		RoleIDToPing: 849690281536389230,
	})
	if err != nil {
		panic(err)
	}
	alerter.Alert(context.Background(), "hello world %v", "error 1 2 3")
}
