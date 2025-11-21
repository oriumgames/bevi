package main

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.1.9

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/df-mc/dragonfly/server"
	"github.com/oriumgames/bevi"
	"github.com/oriumgames/bevi/dragonfly"
)

func main() {
	conf, err := server.DefaultConfig().Config(slog.Default())
	if err != nil {
		panic(err)
	}

	bevi.NewApp().
		AddPlugin(dragonfly.NewPlugin(conf)).
		AddSystems(Systems).
		Run()
}

//bevi:system Update
func DenyBlockBreak(
	srv bevi.Resource[dragonfly.Server],
	r bevi.EventReader[dragonfly.PlayerBlockBreak],
) {
	dragonfly.Receive(srv, r, func(ev dragonfly.PlayerBlockBreak, p *dragonfly.Player) bool {
		p.Message("You can't break blocks here.")
		r.Cancel()
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func WelcomeOnJoin(
	srv bevi.Resource[dragonfly.Server],
	r bevi.EventReader[dragonfly.PlayerJoin],
	f *bevi.Filter1[dragonfly.Player],
) {
	dragonfly.Receive(srv, r, func(ev dragonfly.PlayerJoin, p *dragonfly.Player) bool {
		// Greet the joining player
		p.Message("Welcome to the server! Say \"count\" to see how many players are online.")

		// Announce to everyone else
		q := f.Query()
		for q.Next() {
			c := q.Get()
			if c == p {
				continue
			}
			c.Message(fmt.Sprintf("%s joined the server.", p.Name()))
		}
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func FarewellOnQuit(
	srv bevi.Resource[dragonfly.Server],
	r bevi.EventReader[dragonfly.PlayerQuit],
	f *bevi.Filter1[dragonfly.Player],
) {
	dragonfly.Receive(srv, r, func(ev dragonfly.PlayerQuit, p *dragonfly.Player) bool {
		// Announce to everyone else
		q := f.Query()
		for q.Next() {
			c := q.Get()
			if c == p {
				continue
			}
			c.Message(fmt.Sprintf("%s left the server.", p.Name()))
		}
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func ChatFilterAndCount(
	srv bevi.Resource[dragonfly.Server],
	r bevi.EventReader[dragonfly.PlayerChat],
	f *bevi.Filter1[dragonfly.Player],
) {
	const badWord = "badword" // trivial example; replace with your list or smarter checker

	dragonfly.Receive(srv, r, func(ev dragonfly.PlayerChat, p *dragonfly.Player) bool {
		if ev.Message == nil {
			return true // continue
		}
		msg := *ev.Message
		lmsg := strings.ToLower(msg)

		// Very simple profanity filter
		if strings.Contains(lmsg, badWord) {
			p.Message("Please keep chat clean.")
			r.Cancel()
			return true // continue
		}

		// Respond to "count" message
		if strings.EqualFold(strings.TrimSpace(msg), "count") {
			q := f.Query()
			p.Message(fmt.Sprintf("There are %d players online.", q.Count()))
			// suppress normal chat broadcast by cancelling the event
			r.Cancel()
		}
		return true
	})
}

//bevi:system Update Every=10s
func BroadcastPlayerCount(
	q *bevi.Query1[dragonfly.Player],
) {
	// count first
	n := q.Count()

	// then announce to everyone
	for q.Next() {
		p := q.Get()
		p.Message(fmt.Sprintf("Players online: %d", n))
	}
}
