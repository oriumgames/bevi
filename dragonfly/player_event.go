package dragonfly

import (
	"net"
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
// These map 1:1 to github.com/df-mc/dragonfly/server/player.Handler methods,
// excluding the ctx parameter. The remaining fields match the handler argument
// types exactly and include a ecs.Entity identifying the ECS player entity. // todo

// PlayerMove is a cancellable event and corresponds to HandleMove(ctx *Context, newPos mgl64.Vec3, newRot cube.Rotation).
type PlayerMove struct {
	Player *Player
	NewPos mgl64.Vec3
	NewRot cube.Rotation
}

// PlayerJump corresponds to HandleJump(p *Player).
type PlayerJump struct {
	Player *Player
}

// PlayerTeleport is a cancellable event and corresponds to HandleTeleport(ctx *Context, pos mgl64.Vec3).
type PlayerTeleport struct {
	Player *Player
	Pos    mgl64.Vec3
}

// PlayerChangeWorld corresponds to HandleChangeWorld(p *Player, before, after *world.World).
type PlayerChangeWorld struct {
	Player *Player
	Before *world.World
	After  *world.World
}

// PlayerToggleSprint is a cancellable event and corresponds to HandleToggleSprint(ctx *Context, after bool).
type PlayerToggleSprint struct {
	Player *Player
	After  bool
}

// PlayerToggleSneak is a cancellable event and corresponds to HandleToggleSneak(ctx *Context, after bool).
type PlayerToggleSneak struct {
	Player *Player
	After  bool
}

// PlayerChat is a cancellable event and corresponds to HandleChat(ctx *Context, message *string).
type PlayerChat struct {
	Player  *Player
	Message *string
}

// PlayerFoodLoss is a cancellable event and corresponds to HandleFoodLoss(ctx *Context, from int, to *int).
type PlayerFoodLoss struct {
	Player *Player
	From   int
	To     *int
}

// PlayerHeal is a cancellable event and corresponds to HandleHeal(ctx *Context, health *float64, src world.HealingSource).
type PlayerHeal struct {
	Player *Player
	Health *float64
	Src    world.HealingSource
}

// PlayerHurt is a cancellable event and corresponds to HandleHurt(ctx *Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource).
type PlayerHurt struct {
	Player         *Player
	Damage         *float64
	Immune         bool
	AttackImmunity *time.Duration
	Src            world.DamageSource
}

// PlayerDeath corresponds to HandleDeath(p *Player, src world.DamageSource, keepInv *bool).
type PlayerDeath struct {
	Player  *Player
	Src     world.DamageSource
	KeepInv *bool
}

// PlayerRespawn corresponds to HandleRespawn(p *Player, pos *mgl64.Vec3, w **world.World).
type PlayerRespawn struct {
	Player *Player
	Pos    *mgl64.Vec3
	W      **world.World
}

// PlayerSkinChange is a cancellable event and corresponds to HandleSkinChange(ctx *Context, skin *skin.Skin).
type PlayerSkinChange struct {
	Player *Player
	Skin   *skin.Skin
}

// PlayerFireExtinguish corresponds to HandleFireExtinguish(ctx *Context, pos cube.Pos).
type PlayerFireExtinguish struct {
	Player *Player
	Pos    cube.Pos
}

// PlayerStartBreak is a cancellable event and corresponds to HandleStartBreak(ctx *Context, pos cube.Pos).
type PlayerStartBreak struct {
	Player *Player
	Pos    cube.Pos
}

// PlayerBlockBreak is a cancellable event and corresponds to HandleBlockBreak(ctx *Context, pos cube.Pos, drops *[]item.Stack, xp *int).
type PlayerBlockBreak struct {
	Player *Player
	Pos    cube.Pos
	Drops  *[]item.Stack
	XP     *int
}

// PlayerBlockPlace is a cancellable event and corresponds to HandleBlockPlace(ctx *Context, pos cube.Pos, b world.Block).
type PlayerBlockPlace struct {
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

// PlayerBlockPick is a cancellable event and corresponds to HandleBlockPick(ctx *Context, pos cube.Pos, b world.Block).
type PlayerBlockPick struct {
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

// PlayerItemUse is a cancellable event and corresponds to HandleItemUse(ctx *Context).
type PlayerItemUse struct {
	Player *Player
}

// PlayerItemUseOnBlock is a cancellable event and corresponds to HandleItemUseOnBlock(ctx *Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3).
type PlayerItemUseOnBlock struct {
	Player   *Player
	Pos      cube.Pos
	Face     cube.Face
	ClickPos mgl64.Vec3
}

// PlayerItemUseOnEntity is a cancellable event and corresponds to HandleItemUseOnEntity(ctx *Context, e world.Entity).
type PlayerItemUseOnEntity struct {
	Player *Player
	Target world.Entity
}

// PlayerItemRelease corresponds to HandleItemRelease(ctx *Context, item item.Stack, dur time.Duration).
type PlayerItemRelease struct {
	Player *Player
	Item   item.Stack
	Dur    time.Duration
}

// PlayerItemConsume is a cancellable event and corresponds to HandleItemConsume(ctx *Context, item item.Stack).
type PlayerItemConsume struct {
	Player *Player
	Item   item.Stack
}

// PlayerAttackEntity is a cancellable event and corresponds to HandleAttackEntity(ctx *Context, e world.Entity, force, height *float64, critical *bool).
type PlayerAttackEntity struct {
	Player   *Player
	Target   world.Entity
	Force    *float64
	Height   *float64
	Critical *bool
}

// PlayerExperienceGain is a cancellable event and corresponds to HandleExperienceGain(ctx *Context, amount *int).
type PlayerExperienceGain struct {
	Player *Player
	Amount *int
}

// PlayerPunchAir is a cancellable event and corresponds to HandlePunchAir(ctx *Context).
type PlayerPunchAir struct {
	Player *Player
}

// PlayerSignEdit is a cancellable event and corresponds to HandleSignEdit(ctx *Context, pos cube.Pos, frontSide bool, oldText, newText string).
type PlayerSignEdit struct {
	Player    *Player
	Pos       cube.Pos
	FrontSide bool
	OldText   string
	NewText   string
}

// PlayerLecternPageTurn is a cancellable event and corresponds to HandleLecternPageTurn(ctx *Context, pos cube.Pos, oldPage int, newPage *int).
type PlayerLecternPageTurn struct {
	Player  *Player
	Pos     cube.Pos
	OldPage int
	NewPage *int
}

// PlayerItemDamage is a cancellable event and corresponds to HandleItemDamage(ctx *Context, i item.Stack, damage int).
type PlayerItemDamage struct {
	Player *Player
	Item   item.Stack
	Damage int
}

// PlayerItemPickup is a cancellable event and corresponds to HandleItemPickup(ctx *Context, i *item.Stack).
type PlayerItemPickup struct {
	Player *Player
	Item   *item.Stack
}

// PlayerHeldSlotChange corresponds to HandleHeldSlotChange(ctx *Context, from, to int).
type PlayerHeldSlotChange struct {
	Player *Player
	From   int
	To     int
}

// PlayerItemDrop is a cancellable event and corresponds to HandleItemDrop(ctx *Context, s item.Stack).
type PlayerItemDrop struct {
	Player *Player
	Item   item.Stack
}

// PlayerTransfer is a cancellable event and corresponds to HandleTransfer(ctx *Context, addr *net.UDPAddr).
type PlayerTransfer struct {
	Player *Player
	Addr   *net.UDPAddr
}

// PlayerCommandExecution is a cancellable event and corresponds to HandleCommandExecution(ctx *Context, command cmd.Command, args []string).
type PlayerCommandExecution struct {
	Player  *Player
	Command cmd.Command
	Args    []string
}

// PlayerJoin corresponds to no handler, emitted upon player accept.
type PlayerJoin struct {
	Player *Player
}

// PlayerQuit corresponds to HandleQuit(p *Player).
type PlayerQuit struct {
	Player *Player
}

// PlayerDiagnostics corresponds to HandleDiagnostics(p *Player, d session.Diagnostics).
type PlayerDiagnostics struct {
	Player      *Player
	Diagnostics session.Diagnostics
}

// strictly internal, not for external consumption
type playerCreate struct {
	p *player.Player
}
