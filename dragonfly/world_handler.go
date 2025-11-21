package dragonfly

import (
	"context"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/mlange-42/ark/ecs"
	"github.com/oriumgames/bevi"
)

// worldHandler bridges Dragonfly world events to the ECS and attaches player handlers.
type worldHandler struct {
	ctx   context.Context
	world *ecs.World

	liquidFlow    bevi.EventWriter[WorldLiquidFlow]
	liquidDecay   bevi.EventWriter[WorldLiquidDecay]
	liquidHarden  bevi.EventWriter[WorldLiquidHarden]
	sound         bevi.EventWriter[WorldSound]
	fireSpread    bevi.EventWriter[WorldFireSpread]
	blockBurn     bevi.EventWriter[WorldBlockBurn]
	cropTrample   bevi.EventWriter[WorldCropTrample]
	leavesDecay   bevi.EventWriter[WorldLeavesDecay]
	entitySpawn   bevi.EventWriter[WorldEntitySpawn]
	entityDespawn bevi.EventWriter[WorldEntityDespawn]
	explosion     bevi.EventWriter[WorldExplosion]
	close         bevi.EventWriter[WorldClose]
}

func newWorldHandler(ctx context.Context, app *bevi.App) *worldHandler {
	return &worldHandler{
		ctx:   ctx,
		world: app.World(),

		liquidFlow:    bevi.WriterFor[WorldLiquidFlow](app.Events()),
		liquidDecay:   bevi.WriterFor[WorldLiquidDecay](app.Events()),
		liquidHarden:  bevi.WriterFor[WorldLiquidHarden](app.Events()),
		sound:         bevi.WriterFor[WorldSound](app.Events()),
		fireSpread:    bevi.WriterFor[WorldFireSpread](app.Events()),
		blockBurn:     bevi.WriterFor[WorldBlockBurn](app.Events()),
		cropTrample:   bevi.WriterFor[WorldCropTrample](app.Events()),
		leavesDecay:   bevi.WriterFor[WorldLeavesDecay](app.Events()),
		entitySpawn:   bevi.WriterFor[WorldEntitySpawn](app.Events()),
		entityDespawn: bevi.WriterFor[WorldEntityDespawn](app.Events()),
		explosion:     bevi.WriterFor[WorldExplosion](app.Events()),
		close:         bevi.WriterFor[WorldClose](app.Events()),
	}
}

func (h *worldHandler) HandleLiquidFlow(ctx *world.Context, from cube.Pos, into cube.Pos, liquid world.Liquid, replaced world.Block) {
	if h.liquidFlow.EmitResult(WorldLiquidFlow{
		From:     from,
		Into:     into,
		Liquid:   liquid,
		Replaced: replaced,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleLiquidDecay(ctx *world.Context, pos cube.Pos, before world.Liquid, after world.Liquid) {
	if h.liquidDecay.EmitResult(WorldLiquidDecay{
		Pos:    pos,
		Before: before,
		After:  after,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleLiquidHarden(ctx *world.Context, hardenedPos cube.Pos, liquidHardened world.Block, otherLiquid world.Block, newBlock world.Block) {
	if h.liquidHarden.EmitResult(WorldLiquidHarden{
		HardenedPos:    hardenedPos,
		LiquidHardened: liquidHardened,
		OtherLiquid:    otherLiquid,
		NewBlock:       newBlock,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleSound(ctx *world.Context, s world.Sound, pos mgl64.Vec3) {
	if h.sound.EmitResult(WorldSound{
		S:   s,
		Pos: pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleFireSpread(ctx *world.Context, from cube.Pos, to cube.Pos) {
	if h.fireSpread.EmitResult(WorldFireSpread{
		From: from,
		To:   to,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleBlockBurn(ctx *world.Context, pos cube.Pos) {
	if h.blockBurn.EmitResult(WorldBlockBurn{
		Pos: pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleCropTrample(ctx *world.Context, pos cube.Pos) {
	if h.cropTrample.EmitResult(WorldCropTrample{
		Pos: pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleLeavesDecay(ctx *world.Context, pos cube.Pos) {
	if h.leavesDecay.EmitResult(WorldLeavesDecay{
		Pos: pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleEntitySpawn(tx *world.Tx, entity world.Entity) {
	h.entitySpawn.Emit(WorldEntitySpawn{
		Tx:     tx,
		Entity: entity,
	})
}

func (h *worldHandler) HandleEntityDespawn(tx *world.Tx, entity world.Entity) {
	h.entityDespawn.Emit(WorldEntityDespawn{
		Tx:     tx,
		Entity: entity,
	})
}

func (h *worldHandler) HandleExplosion(ctx *world.Context, position mgl64.Vec3, entities *[]world.Entity, blocks *[]cube.Pos, itemDropChance *float64, spawnFire *bool) {
	if h.explosion.EmitResult(WorldExplosion{
		Position:       position,
		Entities:       entities,
		Blocks:         blocks,
		ItemDropChance: itemDropChance,
		SpawnFire:      spawnFire,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *worldHandler) HandleClose(tx *world.Tx) {
	h.close.Emit(WorldClose{
		Tx: tx,
	})
}
