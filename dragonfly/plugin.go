package dragonfly

//go:generate go run github.com/oriumgames/bevi/cmd/gen@v0.2.0

import (
	"context"

	"github.com/df-mc/dragonfly/server"
	"github.com/go-gl/mathgl/mgl64"
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
					p.Teleport(mgl64.Vec3{0, 14, 0})
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
	srv bevi.Resource[Server],
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
	reader bevi.EventReader[playerRemove],
	writer bevi.EventWriter[PlayerQuit],
) {
	reader.ForEach(func(ev playerRemove) bool {
		writer.Emit(PlayerQuit{
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
	w *bevi.World,
	r bevi.EventReader[playerRemove],
) {
	r.ForEach(func(ev playerRemove) bool {
		w.RemoveEntity(ev.id)
		ev.wg.Done()
		return true
	})
}
