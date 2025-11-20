package bevi

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/oriumgames/ark/ecs"
	"github.com/oriumgames/bevi/internal/event"
	"github.com/oriumgames/bevi/internal/scheduler"
)

// App is the primary entry point for constructing and running a Bevi
// application. It owns the ECS world, the system scheduler, the per-frame
// event bus and the diagnostics adapter. All configuration methods return *App
// to enable chaining before calling Run().
type App struct {
	world  *ecs.World
	sched  *scheduler.Scheduler
	events *event.Bus
	diag   *internalDiagnostics
}

// NewApp constructs a new App with an empty ECS world, a scheduler and a fresh
// event bus. By default a no-op diagnostics implementation is installed.
func NewApp() *App {
	w := ecs.NewWorld()
	bus := event.NewBus()
	sched := scheduler.NewScheduler()
	diag := &internalDiagnostics{
		d: &NopDiagnostics{},
	}

	bus.SetDiagnostics(diag)
	sched.SetDiagnostics(diag)

	return &App{
		world:  &w,
		sched:  sched,
		events: bus,
		diag:   diag,
	}
}

// AddPlugin invokes the given Plugin's Build method, allowing the plugin to
// register systems, resources or other setup. The App is returned for chaining.
func (a *App) AddPlugin(p Plugin) *App {
	p.Build(a)
	return a
}

// AddPlugins invokes AddPlugin for each Plugin in the slice.
// The App is returned for chaining.
func (a *App) AddPlugins(l []Plugin) *App {
	for _, p := range l {
		p.Build(a)
	}
	return a
}

// AddSystem registers a single system function for the specified stage with
// the provided scheduling metadata. The supplied fn must accept (context.Context,
// *ecs.World). The meta.Access field is used to compute parallel batches and
// dependency conflict checks.
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

// AddSystems executes a registration callback that may add multiple systems
// (commonly a generated Systems function). Returns the App for chaining.
func (a *App) AddSystems(reg func(*App)) *App {
	reg(a)
	return a
}

// SetDiagnostics installs an implementation to receive system execution timing
// and error diagnostics. Passing a nil Diagnostics leaves the previous value
// in place (no change). Returns the App for chaining.
func (a *App) SetDiagnostics(d Diagnostics) *App {
	a.diag.d = d
	return a
}

// Run builds the scheduler, then enters the main loop executing stages in
// order. It listens for SIGINT/SIGTERM and cancels the root context to exit.
// Each frame advances events after all Update-stage systems have run.
func (a *App) Run() {
	if err := a.sched.Build(); err != nil {
		log.Fatalf("scheduler build failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer a.sched.Shutdown()

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
	a.events.CompleteNoReader()
	a.events.Advance()

	for {
		if ctx.Err() != nil {
			return
		}
		a.runStage(ctx, PreUpdate)
		a.runStage(ctx, Update)
		a.runStage(ctx, PostUpdate)
		a.events.CompleteNoReader()
		a.events.Advance()
	}
}

func (a *App) runStage(ctx context.Context, stage Stage) {
	a.sched.RunStage(ctx, scheduler.Stage(stage), a.world)
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
