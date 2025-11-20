package dragonfly

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.1.5

import (
	"context"

	"github.com/df-mc/dragonfly/server"
	"github.com/go-gl/mathgl/mgl64"
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
			srv := newServer(p.cfg.New(), w, ecs.NewMap1[Player](w))
			srv.CloseOnProgramEnd()
			srv.Listen()

			go func() {
				h := newPlayerHandler(ctx, app, srv)
				for p := range srv.Accept() {
					p.Handle(h)
					h.HandleJoin(p)
					p.Teleport(mgl64.Vec3{0, 14, 0})
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
		e := w.NewEntity()
		ip := &Player{
			Player: ev.p,
			e:      e,
		}

		mapper.Add(e, ip)
		srv.Get().addPlayer(ip)

		out.Emit(PlayerJoin{
			Entity: e,
		})
		return true
	})
}

// publishPlayerQuit translates the internal playerRemove event into the public PlayerQuit event.
//
//bevi:system PreUpdate Set="dragonfly"
func publishPlayerQuit(
	r bevi.EventReader[playerRemove],
	out bevi.EventWriter[PlayerQuit],
) {
	r.ForEach(func(ev playerRemove) bool {
		out.Emit(PlayerQuit{
			Entity: ev.id,
			wg:     ev.wg,
		})
		return true
	})
}

// handlePlayerRemoval performs the cleanup for a quitting player.
//
//bevi:system PostUpdate Set="dragonfly"
func handlePlayerRemoval(
	w *ecs.World,
	r bevi.EventReader[playerRemove],
) {
	r.ForEach(func(ev playerRemove) bool {
		w.RemoveEntity(ev.id)
		ev.wg.Done()
		return true
	})
}
