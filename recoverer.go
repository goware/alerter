package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

func (a *discordAlerter) Recoverer() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						// we don't recover http.ErrAbortHandler so the response
						// to the client is aborted, this should not be logged
						panic(rvr)
					}

					if !a.SkipLogEntry {
						middleware.PrintPrettyStack(rvr)
					}

					p, err := a.alertWithPanicPayload(r, rvr)
					if err != nil {
						return
					}
					cacheKey := fmt.Sprintf("%d", xxh64FromString(p))
					a.doRequest(r.Context(), cacheKey, p)

					if r.Header.Get("Connection") != "Upgrade" {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

// parseStack is a modified version of parse from go-chi/chi/middleware/recoverer.go
func (a *discordAlerter) parseStack(debugStack []byte, rvr interface{}) (string, error) {
	var err error

	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "panic: %v\n", rvr)

	stack := strings.Split(string(debugStack), "\n")
	lines := []string{}

	// locate panic line, as we may have nested panics
	for i := len(stack) - 1; i > 0; i-- {
		lines = append(lines, stack[i])
		if strings.HasPrefix(stack[i], "panic(") {
			lines = lines[0 : len(lines)-2] // remove boilerplate
			break
		}
	}

	// reverse
	for i := len(lines)/2 - 1; i >= 0; i-- {
		opp := len(lines) - 1 - i
		lines[i], lines[opp] = lines[opp], lines[i]
	}

	for _, l := range lines {
		fmt.Fprintf(buf, "\t %s", l)
	}

	return buf.String(), err
}

func (a *discordAlerter) alertWithPanicPayload(r *http.Request, rvr interface{}) (string, error) {
	debugStack, err := a.parseStack(debug.Stack(), rvr)
	if err != nil {
		return "", err
	}

	p := payload{
		Username:  a.Username,
		AvatarURL: a.AvatarURL,
		Embeds: []embed{
			{
				Author: struct {
					Name    string `json:"name"`
					IconURL string `json:"icon_url"`
				}{Name: a.Username, IconURL: a.AvatarURL},
				Title:       "Panic Stack Trace",
				Description: fmt.Sprintf("```\n %v \n ```", debugStack),
				Fields: []field{
					{
						Name:  "ENV:",
						Value: a.Env,
					},
					{
						Name:  "Request",
						Value: fmt.Sprintf("%s %s", r.Method, r.URL.String()),
					},
				},
				Color: 0xCC0000,
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
