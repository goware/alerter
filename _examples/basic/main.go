package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goware/alerter"
)

func main() {
	alerter, err := alerter.NewDiscordAlerter(&alerter.DiscordConfig{
		WebhookURL: "https://discord.com/api/webhooks/0000000000/abcdefghijklmnopqrstuvwxyz",
		// Username:   "Alerter",
		// AvatarURL:    "https://cdn.discordapp.com/embed/avatars/0.png",
		MentionRoleID: 849690281536389230,
		Env:           "production",
	})
	if err != nil {
		panic(err)
	}
	a.Alert(context.Background(), "hello world %v", "error 1 2 3")

	r := chi.NewRouter()
	r.Use(a.Recoverer())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("oh no")
	})

	http.ListenAndServe(":5555", r)
}
