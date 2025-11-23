package dragonfly

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.2.2

import (
	"context"

	"github.com/df-mc/dragonfly/server"
	"github.com/oriumgames/bevi"
)

// Plugin bridges Dragonfly into Bevi.
type Plugin struct {
	cfg server.Config
}

// NewPlugin constructs a Plugin.
func NewPlugin(cfg server.Config) *Plugin {
	return &Plugin{
		cfg: cfg,
	}
}

func (p *Plugin) Build(app *bevi.App) {
	app.
		AddSystem(bevi.PreStartup, "init", bevi.SystemMeta{
			Set: "dragonfly",
			Access: func() bevi.AccessMeta {
				return bevi.NewAccess()
			}(),
		}, func(ctx context.Context, w *bevi.World) {
			srv := newServer(p.cfg.New(), w, bevi.NewMap1[Player](app))
			srv.CloseOnProgramEnd()
			srv.Listen()

			go func() {
				h := newPlayerHandler(ctx, app, srv)
				for p := range srv.Accept() {
					p.Handle(h)
					h.HandleJoin(p)
				}
			}()

			h := newWorldHandler(ctx, app)
			srv.World().Handle(h)
			srv.Nether().Handle(h)
			srv.End().Handle(h)

			bevi.AddResource(w, srv)
		}).
		AddSystems(Systems)
}

//bevi:system PreUpdate Set="dragonfly"
func emitPlayerJoin(
	w *bevi.World,
	mapper *bevi.Map1[Player],
	srvRes bevi.Resource[Server],
	r bevi.EventReader[playerCreate],
	out bevi.EventWriter[PlayerJoin],
) {
	srv := srvRes.Get()
	r.ForEach(func(ev playerCreate) bool {
		e := w.NewEntity()
		dp := &Player{
			h: ev.p.H(),
			e: e,
			w: srv.World(),

			name: ev.p.Name(),
			xuid: ev.p.XUID(),
			uuid: ev.p.UUID(),
		}

		mapper.Add(e, dp)
		srv.addPlayer(dp)

		out.Emit(PlayerJoin{
			Player: dp,
		})
		return true
	})
}

// publishPlayerQuit translates the internal playerRemove event into the public PlayerQuit event.
//
//bevi:system PreUpdate Set="dragonfly"
func publishPlayerQuit(
	reader bevi.EventReader[playerRemove],
	writer bevi.EventWriter[PlayerQuit],
) {
	reader.ForEach(func(ev playerRemove) bool {
		writer.Emit(PlayerQuit{
			Player: ev.dp,
			wg:     ev.wg,
		})
		return true
	})
}

// handlePlayerRemoval performs the cleanup for a quitting player.
//
//bevi:system PostUpdate Set="dragonfly"
func handlePlayerRemoval(
	w *bevi.World,
	srv bevi.Resource[Server],
	r bevi.EventReader[playerRemove],
) {
	r.ForEach(func(ev playerRemove) bool {
		srv.Get().removePlayer(ev.dp)
		w.RemoveEntity(ev.dp.e)
		ev.wg.Done()
		return true
	})
}
