# bevi

[Bevy](https://docs.rs/bevy_ecs/latest/bevy_ecs/)-inspired ergonomics for [Ark](https://github.com/mlange-42/ark/) ECS: codegen, staged scheduling, and fast typed events.

- Simple runtime: `App` + staged scheduler
- Intelligent parallel scheduler with dependency ordering and access conflict detection
- Per-type, frame-based, high-performance events with cancellation and completion handles
- Code generator that wires your systems together from doc comments and function signatures

## Installation

Add the runtime to your module:

```bash
go get github.com/oriumgames/bevi@v0.1.4
```

Optionally install the generator:

```bash
# As a binary you can call directly (name depends on your shell/OS, shown here via go run)
go install github.com/oriumgames/bevi/cmd/gen@v0.1.4
```

You can also run the generator without installing:

```bash
# From inside this repository or when vendored
go run ./cmd/gen -root .

# From another module (using the latest published version)
go run github.com/oriumgames/bevi/cmd/gen@v0.1.4 -root .
```


## Quick start

1) Define components and annotate your systems:
```go
type Position struct{ X, Y float64 }
type Velocity struct{ X, Y float64 }

//bevi:system Startup
func Spawn(mapper *ecs.Map2[Position, Velocity]) {
    mapper.NewEntity(&Position{X: 0, Y: 0}, &Velocity{X: 1, Y: 0.5})
}

//bevi:system Update Every=16ms
func Move(q *ecs.Query2[Position, Velocity]) {
    for q.Next() {
        p, v := q.Get()
        p.X += v.X
        p.Y += v.Y
    }
}

//bevi:system Update After={"Move"} Every=1s
func PrintCount(q ecs.Query1[Position]) {
    n := 0
    for q.Next() {
        _, n = q.Get(), n+1
    }
    fmt.Println("entities:", n)
}
```

2) Generate glue code:
```bash
go run github.com/oriumgames/bevi/cmd/gen@v0.1.4 -root . -write
```
This writes `bevi_gen.go` next to your files and creates a function:
```go
func Systems(app *bevi.App)
```
that registers all your annotated systems.

3) Boot your app:
```go
func main() {
    bevi.NewApp().
        AddSystems(Systems). // from bevi_gen.go
        Run()
}
```

That’s it. Your app now runs the staged pipeline; systems are ordered, batched for parallelism, throttled by `Every`, and integrated with typed events.


## Writing systems

Bevi uses a single doc-comment line to declare scheduling metadata:

```go
//bevi:system <Stage> [Key=Value ...]
```

Supported keys:
- Stage: one of PreStartup, Startup, PostStartup, PreUpdate, Update, PostUpdate
- Every: Go duration (e.g., `500ms`, `1s`) to throttle execution
- Set: string set/group name (used for Before/After targets as well)
- After: names or set names the system must run after, e.g., `After={"A","B","physics"}`
- Before: names or set names the system must run before
- Reads: component types read (overrides inference)
- Writes: component types written (overrides inference)
- ResReads: resource types read
- ResWrites: resource types written

The generator also infers access from parameters:

- `context.Context` -> passed through
- `*ecs.World` or `ecs.World` -> passed through
- `*ecs.MapN[T...]` -> component WRITE access on T...
- `ecs.QueryN[T...]` -> READ access by default, WRITE access if you accept a pointer `*ecs.QueryN[...]` (write intent marker)
- `*ecs.FilterN[T...]` -> no direct access (it is a builder used to produce queries)
- `ecs.Resource[T]` -> READ access by default, WRITE access if you accept a pointer `*ecs.Resource[T]` (write intent marker)
- `bevi.EventWriter[E]` -> event WRITE access for E
- `bevi.EventReader[E]` -> event READ access for E

The generator synthesizes helpers once per package (mappers, filters, resources, event readers/writers), wires everything in a single `Systems(app *bevi.App)` function. It does not auto-close queries; only call `Close()` yourself when you exit iteration early.

### Filter DSL for queries and filters

You can refine `ecs.FilterN` (and filters used to spawn queries) via extra doc lines:

```go
//bevi:filter <paramName | Qk | Fk> [+Type | -Type | !exclusive | !register]...
```

- `+Type` includes a component type
- `-Type` excludes a component type
- `!exclusive` applies Ark’s `.Exclusive()`
- `!register` applies Ark’s `.Register()`
- Use `Q0`,`Q1` or `F0`,`F1` to refer to positional query/filter parameters if no name is used
- Qualified types may use import aliases; the generator normalizes them

Example:
```go
//bevi:system Update
//bevi:filter q +pkg.Position -pkg.Hidden !exclusive
func Move(q *ecs.Query2[pkg.Position, pkg.Velocity]) { ... }
```


## Generator CLI

```
Usage:
  gen [flags]

Flags:
  -root string          root directory to scan (module/package root) (default ".")
  -write                write generated files (bevi_gen.go); if false, print to stdout (default true)
  -v                    verbose logging to stderr
  -pkg string           only process packages whose name contains this substring
  -include-tests        include _test.go files during scanning
```

Notes:
- The generator writes one `bevi_gen.go` per package that has at least one `//bevi:system` function.
- It skips `bevi_gen.go` itself to avoid feedback loops.
- You can run the generator at any time; it is deterministic and safe to re-run.


## Runtime: App and stages

`bevi.App` orchestrates Ark’s `ecs.World`, the scheduler, and the event bus:

- Stages:
  - PreStartup, Startup, PostStartup (run once at boot)
  - PreUpdate, Update, PostUpdate (run every frame)
- Between stages, the app completes events for frames with no readers and advances the event bus:
  - `events.CompleteNoReader()` then `events.Advance()`

Typical boot:
```go
app := bevi.NewApp().
    AddSystems(Systems).        // from bevi_gen.go
    SetDiagnostics(bevi.NewLogDiagnostics(log.Default()))

app.Run() // blocks until SIGINT/SIGTERM
```

Manual registration (without the generator) is also supported:
```go
acc := bevi.NewAccess()
bevi.AccessWrite[MyComponent](&acc)
meta := bevi.SystemMeta{
    Access: acc,
    After:  []string{"OtherSystem"},
    Every:  250 * time.Millisecond,
}
app.AddSystem(bevi.Update, "MySystem", meta, func(ctx context.Context, w *ecs.World) {
    // ...
})
```


## Scheduler: ordering, conflicts, and parallelism

- Orders systems with a deterministic topological sort using `Before`/`After` constraints.
  - Targets can be system names or `Set` names (applies to all members of that set).
- Builds batches of conflict-free systems to run in parallel.
- Detects access conflicts using precomputed sets and compact bitsets:
  - Component conflicts: write/read, write/write
  - Resource conflicts: write/read, write/write
  - Event conflicts: writer/reader, writer/writer
- Respects `Every` on each system; execution is gated by a high-resolution timestamp.
- Uses a bounded worker pool sized to `GOMAXPROCS` and catches panics, reporting them via diagnostics.


## Events: fast, typed, frame-based

A `bevi.EventBus` delivers events from writers to readers frame-by-frame:

- Writers:
  - `Emit(v T)` fire-and-forget
  - `EmitResult(v T)` returns `EventResult[T]` with completion/cancellation handles
  - `EmitAndWait(ctx, v T)` convenience, returns whether it was cancelled
  - `EmitMany([]T)` bulk emit with fewer allocations

- Readers:
  - `ForEach(func(T) bool)` is the zero-allocation way to iterate events:
    ```go
    reader.ForEach(func(ev MyEvent) bool {
        // optional cancellation
        reader.Cancel()
        if reader.IsCancelled() { /* react */ }
        return true // return false to stop
    })
    ```
  - `Drain()`, `DrainTo(buf)` special cases for batch extraction (when used, writers rely on `CompleteNoReader()` to finalize)

- Results:
  - `Valid()`, `Cancelled()`
  - `Wait(ctx)` blocks until the event finished processing by all readers in the frame
  - `WaitCancelled(ctx)` returns as soon as cancellation is observed, completion, or ctx done

- Frame semantics:
  - Writers append to the “write” buffer this frame.
  - After systems run, the app calls `CompleteNoReader()`, then flips buffers via `Advance()`.
  - Readers iterate the previous frame’s writes.

You can access the bus directly via `app.Events()`, or pass it in context using `bevi.WithEventBus` and fetch typed readers/writers with `bevi.ReaderFromContext[T]` and `bevi.WriterFromContext[T]`.


## Diagnostics

Plug a diagnostics implementation into your app:

```go
type Diagnostics interface {
    SystemStart(name string, stage bevi.Stage)
    SystemEnd(name string, stage bevi.Stage, err error, duration time.Duration)
}

app.SetDiagnostics(bevi.NewLogDiagnostics(log.Default()))
```

Built-ins:
- `NopDiagnostics` – does nothing
- `NewLogDiagnostics(l interface{ Printf(string, ...any) })` – logs start/end and durations, reports panics as errors


## Example

See `./example/test`. It demonstrates:
- Components, events, and multiple `//bevi:system` functions
- Event cancellation and `WaitCancelled`
- Dependencies and `Every` throttling
- Generated `bevi_gen.go` registering all systems


## Tips and gotchas

- Re-run the generator whenever you add/change `//bevi:system` or `//bevi:filter` lines or when parameter types change.
- Pointer-marked queries (`*ecs.QueryN[...]`) are treated as WRITE access; non-pointer queries as READ.
- `Drain()/DrainTo()` don’t register readers; writers will be finalized by `CompleteNoReader()`. Prefer `ForEach()` for normal consumption.
- If you register systems manually, ensure you correctly describe access in `SystemMeta.Access` to unlock safe parallelism.
- If multiple packages contain systems, run the generator once; it will emit a `bevi_gen.go` per package. Call `AddSystems` for each package’s `Systems` function.
- For reliable timing, use `Every` to gate costly systems rather than `time.Sleep` inside the system.


## API surface (selected)

Runtime
- `type App struct`
  - `NewApp() *App`
  - `(*App) AddSystem(stage Stage, name string, meta SystemMeta, fn func(context.Context, *ecs.World)) *App`
  - `(*App) AddSystems(reg func(*App)) *App`
  - `(*App) SetDiagnostics(d Diagnostics) *App`
  - `(*App) Run()`
  - `(*App) World() *ecs.World`
  - `(*App) Events() *EventBus`

Scheduling
- `type Stage int` with: PreStartup, Startup, PostStartup, PreUpdate, Update, PostUpdate
- `type AccessMeta struct` + helpers:
  - `NewAccess() AccessMeta`
  - `AccessRead[T]`, `AccessWrite[T]`, `AccessResRead[T]`, `AccessResWrite[T]`
  - `AccessEventRead[E]`, `AccessEventWrite[E]`
- `type SystemMeta struct { Access AccessMeta; Set string; Before, After []string; Every time.Duration }`

Events
- `type EventBus`
  - `NewEventBus() *EventBus`
  - `(*EventBus) Advance()`
  - `(*EventBus) CompleteNoReader()`
- `WriterFor[T]`, `ReaderFor[T]`
- `type EventWriter[T]`
  - `Emit(T)`, `EmitResult(T) EventResult[T]`, `EmitAndWait(ctx, T) bool`, `EmitMany([]T)`
- `type EventReader[T]`
  - `ForEach(func(T) bool)`, `Cancel()`, `IsCancelled()`, `Drain() []T`, `DrainTo([]T) int`
- `type EventResult[T]`
  - `Valid() bool`, `Cancelled() bool`, `Wait(ctx) bool`, `WaitCancelled(ctx) bool`
- `WithEventBus(ctx, *EventBus) context.Context`, `EventBusFrom(ctx) *EventBus`
- `WriterFromContext[T](ctx) EventWriter[T]`, `ReaderFromContext[T](ctx) EventReader[T]`

Diagnostics
- `type Diagnostics interface`
  - `SystemStart(name string, stage Stage)`
  - `SystemEnd(name string, stage Stage, err error, duration time.Duration)`
- `NopDiagnostics`, `NewLogDiagnostics(logger)`


## License

MIT — see `license.md`.
