package bevi

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mlange-42/ark/ecs"
	"github.com/oriumgames/bevi/internal/event"
	"github.com/oriumgames/bevi/internal/scheduler"
)

type App struct {
	world  *ecs.World
	sched  *scheduler.Scheduler
	events *event.Bus
	diag   *internalDiagnostics
}

func NewApp() *App {
	w := ecs.NewWorld()
	bus := event.NewBus()
	return &App{
		world:  &w,
		sched:  scheduler.NewScheduler(),
		events: bus,
		diag: &internalDiagnostics{
			d: NopDiagnostics{},
		},
	}
}

func (a *App) AddPlugin(p Plugin) *App {
	p.Build(a)
	return a
}

func (a *App) AddPlugins(l []Plugin) *App {
	for _, p := range l {
		p.Build(a)
	}
	return a
}

func (a *App) AddSystem(stage Stage, name string, meta SystemMeta, fn func(context.Context, *ecs.World)) *App {
	sys := &scheduler.System{
		Name:  name,
		Stage: scheduler.Stage(stage),
		Fn: func(ctx context.Context, w any) {
			fn(ctx, w.(*ecs.World))
		},
		Meta: meta.toInternal(),
	}
	a.sched.AddSystem(sys)
	return a
}

func (a *App) AddSystems(reg func(*App)) *App {
	reg(a)
	return a
}

func (a *App) Run() {
	if err := a.sched.Build(); err != nil {
		log.Fatalf("scheduler build failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	go func() {
		<-sig
		cancel()
	}()

	a.runStage(ctx, PreStartup)
	a.runStage(ctx, Startup)
	a.runStage(ctx, PostStartup)

	for {
		if ctx.Err() != nil {
			return
		}
		a.runStage(ctx, PreUpdate)
		a.runStage(ctx, Update)
		a.runStage(ctx, PostUpdate)
	}
}

func (a *App) runStage(ctx context.Context, stage Stage) {
	a.sched.RunStage(ctx, scheduler.Stage(stage), a.world, a.diag)
	a.events.CompleteNoReader()
	a.events.Advance()
}

func (a *App) World() *ecs.World {
	return a.world
}

func (a *App) Events() *event.Bus {
	return a.events
}

type Plugin interface {
	Build(app *App)
}
