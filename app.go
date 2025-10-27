package bevi

import (
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

func (a *App) World() *ecs.World {
	return a.world
}

type Plugin interface {
	Build(app *App)
}
