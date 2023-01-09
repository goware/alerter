package main

import (
	"context"

	"github.com/goware/alerter"
)

func main() {
	discordAlerter, err := alerter.NewDiscordAlerter(&alerter.DiscordConfig{
		// WebhookURL: "https://discord.com/api/webhooks/0000000000/abcdefghijklmnopqrstuvwxyz",
		WebhookURL: "https://discord.com/api/webhooks/1026489049168486542/o-PdzNFZYFkeUa5srodaufU15lQ4DOrtYKq9kVuRbIASNZBmEBuPvFNEjIUn9NFInHP4",
		// Username:   "Alerter",
		// AvatarURL:    "https://cdn.discordapp.com/embed/avatars/0.png",
		RoleIDToPing: 849690281536389230,
	})
	if err != nil {
		panic(err)
	}
	discordAlerter.Alert(context.Background(), alerter.LevelError, "hello world %v", "error 1 2 3")
}
