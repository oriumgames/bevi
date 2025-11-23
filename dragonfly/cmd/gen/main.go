package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// EventDesc describes a single event method.
type EventDesc struct {
	Name        string
	Params      []Param
	Cancellable bool
	Context     string // "ctx *player.Context" or "p *player.Player" or "tx *world.Tx"
}

type Param struct {
	Name string
	Type string
}

// Domains
var playerEvents = []EventDesc{
	{Name: "Move", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"newPos", "mgl64.Vec3"}, {"newRot", "cube.Rotation"}}},
	{Name: "Jump", Cancellable: false, Context: "p *player.Player", Params: []Param{}},
	{Name: "Teleport", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "mgl64.Vec3"}}},
	{Name: "ChangeWorld", Cancellable: false, Context: "p *player.Player", Params: []Param{{"before", "*world.World"}, {"after", "*world.World"}}},
	{Name: "ToggleSprint", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"after", "bool"}}},
	{Name: "ToggleSneak", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"after", "bool"}}},
	{Name: "Chat", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"message", "*string"}}},
	{Name: "FoodLoss", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"from", "int"}, {"to", "*int"}}},
	{Name: "Heal", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"health", "*float64"}, {"src", "world.HealingSource"}}},
	{Name: "Hurt", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"damage", "*float64"}, {"immune", "bool"}, {"attackImmunity", "*time.Duration"}, {"src", "world.DamageSource"}}},
	{Name: "Death", Cancellable: false, Context: "p *player.Player", Params: []Param{{"src", "world.DamageSource"}, {"keepInv", "*bool"}}},
	{Name: "Respawn", Cancellable: false, Context: "p *player.Player", Params: []Param{{"pos", "*mgl64.Vec3"}, {"w", "**world.World"}}},
	{Name: "SkinChange", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"skin", "*skin.Skin"}}},
	{Name: "FireExtinguish", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}}},
	{Name: "StartBreak", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}}},
	{Name: "BlockBreak", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}, {"drops", "*[]item.Stack"}, {"xp", "*int"}}},
	{Name: "BlockPlace", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}, {"block", "world.Block"}}},
	{Name: "BlockPick", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}, {"block", "world.Block"}}},
	{Name: "ItemUse", Cancellable: true, Context: "ctx *player.Context", Params: []Param{}},
	{Name: "ItemUseOnBlock", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}, {"face", "cube.Face"}, {"clickPos", "mgl64.Vec3"}}},
	{Name: "ItemUseOnEntity", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"target", "world.Entity"}}},
	{Name: "ItemRelease", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"item", "item.Stack"}, {"dur", "time.Duration"}}},
	{Name: "ItemConsume", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"item", "item.Stack"}}},
	{Name: "AttackEntity", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"target", "world.Entity"}, {"force", "*float64"}, {"height", "*float64"}, {"critical", "*bool"}}},
	{Name: "ExperienceGain", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"amount", "*int"}}},
	{Name: "PunchAir", Cancellable: true, Context: "ctx *player.Context", Params: []Param{}},
	{Name: "SignEdit", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}, {"frontSide", "bool"}, {"oldText", "string"}, {"newText", "string"}}},
	{Name: "LecternPageTurn", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"pos", "cube.Pos"}, {"oldPage", "int"}, {"newPage", "*int"}}},
	{Name: "ItemDamage", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"item", "item.Stack"}, {"damage", "int"}}},
	{Name: "ItemPickup", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"item", "*item.Stack"}}},
	{Name: "HeldSlotChange", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"from", "int"}, {"to", "int"}}},
	{Name: "ItemDrop", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"item", "item.Stack"}}},
	{Name: "Transfer", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"addr", "*net.UDPAddr"}}},
	{Name: "CommandExecution", Cancellable: true, Context: "ctx *player.Context", Params: []Param{{"command", "cmd.Command"}, {"args", "[]string"}}},
	{Name: "Join", Cancellable: false, Context: "p *player.Player", Params: []Param{}},
	{Name: "Quit", Cancellable: false, Context: "p *player.Player", Params: []Param{}},
	{Name: "Diagnostics", Cancellable: false, Context: "p *player.Player", Params: []Param{{"diagnostics", "session.Diagnostics"}}},
}

var worldEvents = []EventDesc{
	{Name: "LiquidFlow", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"from", "cube.Pos"}, {"into", "cube.Pos"}, {"liquid", "world.Liquid"}, {"replaced", "world.Block"}}},
	{Name: "LiquidDecay", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"pos", "cube.Pos"}, {"before", "world.Liquid"}, {"after", "world.Liquid"}}},
	{Name: "LiquidHarden", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"hardenedPos", "cube.Pos"}, {"liquidHardened", "world.Block"}, {"otherLiquid", "world.Block"}, {"newBlock", "world.Block"}}},
	{Name: "Sound", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"s", "world.Sound"}, {"pos", "mgl64.Vec3"}}},
	{Name: "FireSpread", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"from", "cube.Pos"}, {"to", "cube.Pos"}}},
	{Name: "BlockBurn", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"pos", "cube.Pos"}}},
	{Name: "CropTrample", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"pos", "cube.Pos"}}},
	{Name: "LeavesDecay", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"pos", "cube.Pos"}}},
	{Name: "EntitySpawn", Cancellable: false, Context: "tx *world.Tx", Params: []Param{{"entity", "world.Entity"}}},
	{Name: "EntityDespawn", Cancellable: false, Context: "tx *world.Tx", Params: []Param{{"entity", "world.Entity"}}},
	{Name: "Explosion", Cancellable: true, Context: "ctx *world.Context", Params: []Param{{"position", "mgl64.Vec3"}, {"entities", "*[]world.Entity"}, {"blocks", "*[]cube.Pos"}, {"itemDropChance", "*float64"}, {"spawnFire", "*bool"}}},
	{Name: "Close", Cancellable: false, Context: "tx *world.Tx", Params: []Param{}},
}

func main() {
	if err := genPlayerEvents("./player_event.go"); err != nil {
		panic(err)
	}
	if err := genPlayerHandler("./player_handler.go"); err != nil {
		panic(err)
	}
	if err := genWorldEvents("./world_event.go"); err != nil {
		panic(err)
	}
	if err := genWorldHandler("./world_handler.go"); err != nil {
		panic(err)
	}
}

func toExported(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// --- Player Generation ---

func genPlayerEvents(path string) error {
	tmpl := `package dragonfly

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Player events.
// These map 1:1 to github.com/df-mc/dragonfly/server/player.Handler methods.

type PlayerEvent interface {
	PlayerRef() *Player
}

{{range $ev := .}}
// Player{{$ev.Name}} {{if $ev.Cancellable}}is a cancellable event and {{end}}corresponds to Handle{{$ev.Name}}({{$ev.Context}}{{range $ev.Params}}, {{.Name}} {{.Type}}{{end}}).
type Player{{$ev.Name}} struct {
    Player *Player
{{- range $p := $ev.Params}}
    {{- if and (eq $ev.Name "Death") (eq $p.Name "keepInv")}}
    KeepInv *atomic.Bool
    {{- else}}
    {{$p.Name | toExported}} {{$p.Type}}
    {{- end}}
{{- end}}
{{- if eq $ev.Name "Quit"}}
    wg *sync.WaitGroup
{{- end}}
}

func (p Player{{$ev.Name}}) PlayerRef() *Player { return p.Player }
{{end}}

// PreQuit is special
type PlayerPreQuit struct {
	Player *Player
}

func (p PlayerPreQuit) PlayerRef() *Player { return p.Player }

// strictly internal, not for external consumption
type playerCreate struct {
	p *player.Player
}

type playerRemove struct {
	dp *Player
	wg *sync.WaitGroup
}
`
	return render(path, tmpl, playerEvents)
}

func genPlayerHandler(path string) error {
	tmpl := `package dragonfly

import (
    "context"
    "net"
    "sync"
    "sync/atomic"
    "time"

    "github.com/df-mc/dragonfly/server/block/cube"
    "github.com/df-mc/dragonfly/server/cmd"
    "github.com/df-mc/dragonfly/server/item"
    "github.com/df-mc/dragonfly/server/player"
    "github.com/df-mc/dragonfly/server/player/skin"
    "github.com/df-mc/dragonfly/server/session"
    "github.com/df-mc/dragonfly/server/world"
    "github.com/go-gl/mathgl/mgl64"
    "github.com/oriumgames/bevi"
)

// playerHandler bridges Dragonfly player events to the ECS.
type playerHandler struct {
    ctx    context.Context
    srv    *Server
    world  *bevi.World

    keepInv atomic.Bool

{{range .}}
    {{.Name | lowerFirst}} bevi.EventWriter[Player{{.Name}}]
{{- end}}
    preQuit bevi.EventWriter[PlayerPreQuit]

    // internal
    create bevi.EventWriter[playerCreate]
    remove bevi.EventWriter[playerRemove]
}

func newPlayerHandler(ctx context.Context, app *bevi.App, srv *Server) *playerHandler {
    return &playerHandler{
        ctx:    ctx,
        srv:    srv,
        world:  app.World(),

{{range .}}
        {{.Name | lowerFirst}}: bevi.WriterFor[Player{{.Name}}](app.Events()),
{{- end}}
        preQuit: bevi.WriterFor[PlayerPreQuit](app.Events()),

        // internal
        create: bevi.WriterFor[playerCreate](app.Events()),
        remove: bevi.WriterFor[playerRemove](app.Events()),
    }
}

{{range .}}
func (h *playerHandler) Handle{{.Name}}({{.Context}}{{range .Params}}, {{.Name}} {{.Type}}{{end}}) {

{{- if eq .Name "Hurt"}}
    // Custom Hurt override
    dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
    if !ok {
        return
    }

    // Fire Hurt event
    if h.hurt.EmitResult(PlayerHurt{
        Player:         dp,
        Damage:         damage,
        Immune:         immune,
        AttackImmunity: attackImmunity,
        Src:            src,
    }).Wait(h.ctx) {
        ctx.Cancel()
    }

    // Custom death trigger
    if (ctx.Val().Health() - *damage) > 0 {
        return
    }

    if h.death.EmitResult(PlayerDeath{
        Player:  dp,
        Src:     src,
        KeepInv: &h.keepInv,
    }).Wait(h.ctx) {
        ctx.Cancel()
    }

{{- else if eq .Name "Death"}}
    // Custom Death override
    *keepInv = h.keepInv.Load()

{{- else if eq .Name "Join"}}
    h.create.Emit(playerCreate{
        p: p,
    })

{{- else if eq .Name "Quit"}}
    dp, ok := h.srv.PlayerByUUID(p.UUID())
    if !ok {
        return
    }

    h.preQuit.Emit(PlayerPreQuit{
        Player: dp,
    })

    var wg sync.WaitGroup
    wg.Add(1)

    h.remove.Emit(playerRemove{
        dp: dp,
        wg: &wg,
    })

    wg.Wait()

{{- else}}
    {{- if hasPrefix .Context "p *player.Player"}}
    dp, ok := h.srv.PlayerByUUID(p.UUID())
    {{- else}}
    dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
    {{- end}}
    if !ok {
        return
    }
    {{- if .Cancellable}}
    if h.{{.Name | lowerFirst}}.EmitResult(Player{{.Name}}{
        Player: dp,
    {{- range .Params}}
        {{.Name | toExported}}: {{.Name}},
    {{- end}}
    }).Wait(h.ctx) {
        ctx.Cancel()
    }
    {{- else}}
    h.{{.Name | lowerFirst}}.Emit(Player{{.Name}}{
        Player: dp,
    {{- range .Params}}
        {{.Name | toExported}}: {{.Name}},
    {{- end}}
    })
    {{- end}}
{{- end}}
}
{{end}}
`
	return render(path, tmpl, playerEvents)
}

// --- World Generation ---

func genWorldEvents(path string) error {
	tmpl := `package dragonfly

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// World events.
// These map 1:1 to github.com/df-mc/dragonfly/server/world.Handler methods,
// excluding the ctx parameter for context-carrying callbacks. Argument types
// match exactly.

{{range .}}
// World{{.Name}} {{if .Cancellable}}is a cancellable event and {{end}}corresponds to Handle{{.Name}}({{.Context}}{{range .Params}}, {{.Name}} {{.Type}}{{end}}).
type World{{.Name}} struct {
{{- if or (eq .Name "EntitySpawn") (eq .Name "EntityDespawn") (eq .Name "Close")}}
	Tx *world.Tx
{{- end}}
{{- range .Params}}
	{{.Name | toExported}} {{.Type}}
{{- end}}
}
{{end}}
`
	return render(path, tmpl, worldEvents)
}

func genWorldHandler(path string) error {
	tmpl := `package dragonfly

import (
	"context"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/oriumgames/bevi"
)

// worldHandler bridges Dragonfly world events to the ECS and attaches player handlers.
type worldHandler struct {
	ctx   context.Context
	world *bevi.World

{{range .}}
	{{.Name | lowerFirst}} bevi.EventWriter[World{{.Name}}]
{{- end}}
}

func newWorldHandler(ctx context.Context, app *bevi.App) *worldHandler {
	return &worldHandler{
		ctx:   ctx,
		world: app.World(),

{{range .}}
		{{.Name | lowerFirst}}: bevi.WriterFor[World{{.Name}}](app.Events()),
{{- end}}
	}
}

{{range .}}
func (h *worldHandler) Handle{{.Name}}({{.Context}}{{range .Params}}, {{.Name}} {{.Type}}{{end}}) {
	{{- if .Cancellable}}
	if h.{{.Name | lowerFirst}}.EmitResult(World{{.Name}}{
	{{- range .Params}}
		{{.Name | toExported}}: {{.Name}},
	{{- end}}
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
	{{- else}}
	h.{{.Name | lowerFirst}}.Emit(World{{.Name}}{
	{{- if or (eq .Name "EntitySpawn") (eq .Name "EntityDespawn") (eq .Name "Close")}}
		Tx: tx,
	{{- end}}
	{{- range .Params}}
		{{.Name | toExported}}: {{.Name}},
	{{- end}}
	})
	{{- end}}
}
{{end}}
`
	return render(path, tmpl, worldEvents)
}

// --- Helpers ---

func render(path string, tmplStr string, data any) error {
	funcs := template.FuncMap{
		"toExported": toExported,
		"lowerFirst": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
		"hasPrefix": strings.HasPrefix,
		"eq":        func(a, b string) bool { return a == b },
	}

	t, err := template.New("").Funcs(funcs).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format source: %w\n%s", err, buf.String())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, src, 0644)
}
