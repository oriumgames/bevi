package bevi

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mlange-42/ark/ecs"
	"github.com/oriumgames/bevi/internal/scheduler"
)

type App struct {
	world *ecs.World
	sched *scheduler.Scheduler
}

func NewApp() *App {
	w := ecs.NewWorld()
	return &App{
		world: &w,
		sched: scheduler.NewScheduler(),
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

	a.sched.RunStage(ctx, scheduler.Stage(Startup), a.world, nil)
	for {
		if ctx.Err() != nil {
			return
		}
		a.sched.RunStage(ctx, scheduler.Stage(Update), a.world, nil)
	}
}

func (a *App) World() *ecs.World {
	return a.world
}

type Plugin interface {
	Build(app *App)
}
