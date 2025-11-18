package dragonfly

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.1.1

import (
	"context"

	"github.com/df-mc/dragonfly/server"
	"github.com/oriumgames/ark/ecs"
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
		}, func(ctx context.Context, w *ecs.World) {
			srv := newServer(p.cfg.New())
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

			ecs.AddResource(w, srv)
		}).
		AddSystems(Systems)
}

//bevi:system PreUpdate Set="dragonfly"
func emitPlayerJoin(
	w *ecs.World,
	mapper *ecs.Map1[Player],
	srv ecs.Resource[Server],
	r bevi.EventReader[playerCreate],
	out bevi.EventWriter[PlayerJoin],
) {
	r.ForEach(func(ev playerCreate) bool {
		id := ev.p.UUID()
		if _, ok := srv.Get().Player(id); ok {
			return true
		}

		e := w.NewEntity()
		ip := &Player{
			Player: ev.p,
			e:      e,
		}

		mapper.Add(e, ip)
		srv.Get().addPlayer(ip)

		out.Emit(PlayerJoin{
			Player: ip,
		})
		return true
	})
}

//bevi:system PreUpdate Set="dragonfly"
func emitPlayerQuit(
	w *ecs.World,
	srv ecs.Resource[Server],
	r bevi.EventReader[playerRemove],
	out bevi.EventWriter[PlayerQuit],
) {
	r.ForEach(func(ev playerRemove) bool {
		ip, ok := srv.Get().Player(ev.id)
		if !ok {
			return true
		}

		out.Emit(PlayerQuit{
			Player: ip,
		})

		srv.Get().removePlayer(ip)
		w.RemoveEntity(ip.e)
		return true
	})
}
