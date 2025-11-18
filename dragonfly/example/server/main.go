package main

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.1.4

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/df-mc/dragonfly/server"
	"github.com/oriumgames/ark/ecs"
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
	r bevi.EventReader[dragonfly.PlayerBlockBreak],
) {
	r.ForEach(func(ev dragonfly.PlayerBlockBreak) bool {
		ev.Player.Message("You can't break blocks here.")
		r.Cancel()
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func WelcomeOnJoin(
	r bevi.EventReader[dragonfly.PlayerJoin],
	f *ecs.Filter1[dragonfly.Player],
) {
	r.ForEach(func(ev dragonfly.PlayerJoin) bool {
		// Greet the joining player
		ev.Player.Message("Welcome to the server! Say \"count\" to see how many players are online.")

		// Announce to everyone else
		q := f.Query()
		for q.Next() {
			p := q.Get()
			if p == ev.Player {
				continue
			}
			p.Message(fmt.Sprintf("%s joined the server.", ev.Player.Name()))
		}
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func FarewellOnQuit(
	r bevi.EventReader[dragonfly.PlayerQuit],
	f *ecs.Filter1[dragonfly.Player],
) {
	r.ForEach(func(ev dragonfly.PlayerQuit) bool {
		// Announce to everyone else
		q := f.Query()
		for q.Next() {
			p := q.Get()
			if p == ev.Player {
				continue
			}
			p.Message(fmt.Sprintf("%s left the server.", ev.Player.Name()))
		}
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func ChatFilterAndCount(
	r bevi.EventReader[dragonfly.PlayerChat],
	f *ecs.Filter1[dragonfly.Player],
) {
	const badWord = "badword" // trivial example; replace with your list or smarter checker

	r.ForEach(func(ev dragonfly.PlayerChat) bool {
		if ev.Message == nil {
			return true // continue
		}
		msg := *ev.Message
		lmsg := strings.ToLower(msg)

		// Very simple profanity filter
		if strings.Contains(lmsg, badWord) {
			ev.Player.Message("Please keep chat clean.")
			r.Cancel()
			return true // continue
		}

		// Respond to "count" message
		if strings.EqualFold(strings.TrimSpace(msg), "count") {
			q := f.Query()
			ev.Player.Message(fmt.Sprintf("There are %d players online.", q.Count()))
			// suppress normal chat broadcast by cancelling the event
			r.Cancel()
		}
		return true
	})
}

//bevi:system Update Every=10s
func BroadcastPlayerCount(
	q *ecs.Query1[dragonfly.Player],
) {
	// count first
	n := q.Count()

	// then announce to everyone
	for q.Next() {
		p := q.Get()
		p.Message(fmt.Sprintf("Players online: %d", n))
	}
}
