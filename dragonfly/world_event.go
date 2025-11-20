package dragonfly

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// World events.
// These map 1:1 to github.com/df-mc/dragonfly/server/world.Handler methods,
// excluding the ctx parameter for context-carrying callbacks. Argument types
// match exactly.

// WorldLiquidFlow is a cancellable event and corresponds to HandleLiquidFlow(ctx *world.Context, from cube.Pos, into cube.Pos, liquid world.Liquid, replaced world.Block).
type WorldLiquidFlow struct {
	From     cube.Pos
	Into     cube.Pos
	Liquid   world.Liquid
	Replaced world.Block
}

// WorldLiquidDecay is a cancellable event and corresponds to HandleLiquidDecay(ctx *world.Context, pos cube.Pos, before world.Liquid, after world.Liquid).
type WorldLiquidDecay struct {
	Pos    cube.Pos
	Before world.Liquid
	After  world.Liquid
}

// WorldLiquidHarden is a cancellable event and corresponds to HandleLiquidHarden(ctx *world.Context, hardenedPos cube.Pos, liquidHardened world.Block, otherLiquid world.Block, newBlock world.Block).
type WorldLiquidHarden struct {
	HardenedPos    cube.Pos
	LiquidHardened world.Block
	OtherLiquid    world.Block
	NewBlock       world.Block
}

// WorldSound is a cancellable event and corresponds to HandleSound(ctx *world.Context, s world.Sound, pos mgl64.Vec3).
type WorldSound struct {
	S   world.Sound
	Pos mgl64.Vec3
}

// WorldFireSpread is a cancellable event and corresponds to HandleFireSpread(ctx *world.Context, from cube.Pos, to cube.Pos).
type WorldFireSpread struct {
	From cube.Pos
	To   cube.Pos
}

// WorldBlockBurn is a cancellable event and corresponds to HandleBlockBurn(ctx *world.Context, pos cube.Pos).
type WorldBlockBurn struct {
	Pos cube.Pos
}

// WorldCropTrample is a cancellable event and corresponds to HandleCropTrample(ctx *world.Context, pos cube.Pos).
type WorldCropTrample struct {
	Pos cube.Pos
}

// WorldLeavesDecay is a cancellable event and corresponds to HandleLeavesDecay(ctx *world.Context, pos cube.Pos).
type WorldLeavesDecay struct {
	Pos cube.Pos
}

// WorldEntitySpawn corresponds to HandleEntitySpawn(tx *world.Tx, entity world.Entity).
type WorldEntitySpawn struct {
	Tx     *world.Tx
	Entity world.Entity
}

// WorldEntityDespawn corresponds to HandleEntityDespawn(tx *world.Tx, entity world.Entity).
type WorldEntityDespawn struct {
	Tx     *world.Tx
	Entity world.Entity
}

// WorldExplosion is a cancellable event and corresponds to HandleExplosion(ctx *world.Context, position mgl64.Vec3, entities *[]world.Entity, blocks *[]cube.Pos, itemDropChance *float64, spawnFire *bool).
type WorldExplosion struct {
	Position       mgl64.Vec3
	Entities       *[]world.Entity
	Blocks         *[]cube.Pos
	ItemDropChance *float64
	SpawnFire      *bool
}

// WorldClose corresponds to HandleClose(tx *world.Tx).
type WorldClose struct {
	Tx *world.Tx
}
