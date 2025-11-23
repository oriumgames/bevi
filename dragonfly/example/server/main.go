package main

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.2.1

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/oriumgames/bevi"
	"github.com/oriumgames/bevi/dragonfly"
)

func main() {
	cfg := server.DefaultConfig()
	cfg.Network.Address = ":19135"
	conf, err := cfg.Config(slog.Default())
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
	w *bevi.World,
	r bevi.EventReader[dragonfly.PlayerBlockBreak],
) {
	dragonfly.Receive(w, &r, func(ev dragonfly.PlayerBlockBreak) bool {
		ev.Player.Exec(func(tx *world.Tx, p *player.Player) {
			p.Message("You can't break blocks here.")
		})
		r.Cancel()
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func WelcomeOnJoin(
	w *bevi.World,
	r bevi.EventReader[dragonfly.PlayerJoin],
	f *bevi.Filter1[dragonfly.Player],
) {
	dragonfly.Receive(w, &r, func(ev dragonfly.PlayerJoin) bool {
		// Greet the joining player
		ev.Player.Exec(func(tx *world.Tx, p *player.Player) {
			p.Message("Welcome to the server! Say \"count\" to see how many players are online.")
		})

		// Announce to everyone else
		q := f.Query()
		for q.Next() {
			c := q.Get()
			if c == ev.Player {
				continue
			}
			c.Exec(func(tx *world.Tx, p *player.Player) {
				p.Message(fmt.Sprintf("%s joined the server.", ev.Player.Name()))
			})
		}
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func FarewellOnQuit(
	w *bevi.World,
	r bevi.EventReader[dragonfly.PlayerQuit],
	f *bevi.Filter1[dragonfly.Player],
) {
	dragonfly.Receive(w, &r, func(ev dragonfly.PlayerQuit) bool {
		// Announce to everyone else
		q := f.Query()
		for q.Next() {
			c := q.Get()
			if c == ev.Player {
				continue
			}
			c.Exec(func(tx *world.Tx, c *player.Player) {
				c.Message(fmt.Sprintf("%s left the server.", ev.Player.Name()))
			})
		}
		return true
	})
}

//bevi:system Update Reads={dragonfly.Player}
func ChatFilterAndCount(
	w *bevi.World,
	r bevi.EventReader[dragonfly.PlayerChat],
	f *bevi.Filter1[dragonfly.Player],
) {
	const badWord = "badword" // trivial example; replace with your list or smarter checker

	dragonfly.Receive(w, &r, func(ev dragonfly.PlayerChat) bool {
		if ev.Message == nil {
			return true // continue
		}
		msg := *ev.Message
		lmsg := strings.ToLower(msg)

		// Very simple profanity filter
		if strings.Contains(lmsg, badWord) {
			ev.Player.Exec(func(tx *world.Tx, p *player.Player) {
				p.Message("Please keep chat clean.")
			})
			r.Cancel()
			return true // continue
		}

		// Respond to "count" message
		if strings.EqualFold(strings.TrimSpace(msg), "count") {
			q := f.Query()
			ev.Player.Exec(func(tx *world.Tx, p *player.Player) {
				p.Message(fmt.Sprintf("There are %d players online.", q.Count()))
			})

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
		dp := q.Get()
		dp.Exec(func(tx *world.Tx, p *player.Player) {
			p.Message(fmt.Sprintf("Players online: %d", n))
		})
	}
}
