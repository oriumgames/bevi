package dragonfly

import (
	"net"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/oriumgames/ark/ecs"
)

// Player events.
// These map 1:1 to github.com/df-mc/dragonfly/server/player.Handler methods.

type PlayerEvent interface {
	Player() ecs.Entity
}

// PlayerMove is a cancellable event and corresponds to HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation).
type PlayerMove struct {
	Entity ecs.Entity
	NewPos mgl64.Vec3
	NewRot cube.Rotation
}

func (p PlayerMove) Player() ecs.Entity { return p.Entity }

// PlayerJump corresponds to HandleJump(p *player.Player).
type PlayerJump struct {
	Entity ecs.Entity
}

func (p PlayerJump) Player() ecs.Entity { return p.Entity }

// PlayerTeleport is a cancellable event and corresponds to HandleTeleport(ctx *player.Context, pos mgl64.Vec3).
type PlayerTeleport struct {
	Entity ecs.Entity
	Pos    mgl64.Vec3
}

func (p PlayerTeleport) Player() ecs.Entity { return p.Entity }

// PlayerChangeWorld corresponds to HandleChangeWorld(p *player.Player, before *world.World, after *world.World).
type PlayerChangeWorld struct {
	Entity ecs.Entity
	Before *world.World
	After  *world.World
}

func (p PlayerChangeWorld) Player() ecs.Entity { return p.Entity }

// PlayerToggleSprint is a cancellable event and corresponds to HandleToggleSprint(ctx *player.Context, after bool).
type PlayerToggleSprint struct {
	Entity ecs.Entity
	After  bool
}

func (p PlayerToggleSprint) Player() ecs.Entity { return p.Entity }

// PlayerToggleSneak is a cancellable event and corresponds to HandleToggleSneak(ctx *player.Context, after bool).
type PlayerToggleSneak struct {
	Entity ecs.Entity
	After  bool
}

func (p PlayerToggleSneak) Player() ecs.Entity { return p.Entity }

// PlayerChat is a cancellable event and corresponds to HandleChat(ctx *player.Context, message *string).
type PlayerChat struct {
	Entity  ecs.Entity
	Message *string
}

func (p PlayerChat) Player() ecs.Entity { return p.Entity }

// PlayerFoodLoss is a cancellable event and corresponds to HandleFoodLoss(ctx *player.Context, from int, to *int).
type PlayerFoodLoss struct {
	Entity ecs.Entity
	From   int
	To     *int
}

func (p PlayerFoodLoss) Player() ecs.Entity { return p.Entity }

// PlayerHeal is a cancellable event and corresponds to HandleHeal(ctx *player.Context, health *float64, src world.HealingSource).
type PlayerHeal struct {
	Entity ecs.Entity
	Health *float64
	Src    world.HealingSource
}

func (p PlayerHeal) Player() ecs.Entity { return p.Entity }

// PlayerHurt is a cancellable event and corresponds to HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource).
type PlayerHurt struct {
	Entity         ecs.Entity
	Damage         *float64
	Immune         bool
	AttackImmunity *time.Duration
	Src            world.DamageSource
}

func (p PlayerHurt) Player() ecs.Entity { return p.Entity }

// PlayerDeath corresponds to HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool).
type PlayerDeath struct {
	Entity  ecs.Entity
	Src     world.DamageSource
	KeepInv *bool
}

func (p PlayerDeath) Player() ecs.Entity { return p.Entity }

// PlayerRespawn corresponds to HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World).
type PlayerRespawn struct {
	Entity ecs.Entity
	Pos    *mgl64.Vec3
	W      **world.World
}

func (p PlayerRespawn) Player() ecs.Entity { return p.Entity }

// PlayerSkinChange is a cancellable event and corresponds to HandleSkinChange(ctx *player.Context, skin *skin.Skin).
type PlayerSkinChange struct {
	Entity ecs.Entity
	Skin   *skin.Skin
}

func (p PlayerSkinChange) Player() ecs.Entity { return p.Entity }

// PlayerFireExtinguish is a cancellable event and corresponds to HandleFireExtinguish(ctx *player.Context, pos cube.Pos).
type PlayerFireExtinguish struct {
	Entity ecs.Entity
	Pos    cube.Pos
}

func (p PlayerFireExtinguish) Player() ecs.Entity { return p.Entity }

// PlayerStartBreak is a cancellable event and corresponds to HandleStartBreak(ctx *player.Context, pos cube.Pos).
type PlayerStartBreak struct {
	Entity ecs.Entity
	Pos    cube.Pos
}

func (p PlayerStartBreak) Player() ecs.Entity { return p.Entity }

// PlayerBlockBreak is a cancellable event and corresponds to HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int).
type PlayerBlockBreak struct {
	Entity ecs.Entity
	Pos    cube.Pos
	Drops  *[]item.Stack
	Xp     *int
}

func (p PlayerBlockBreak) Player() ecs.Entity { return p.Entity }

// PlayerBlockPlace is a cancellable event and corresponds to HandleBlockPlace(ctx *player.Context, pos cube.Pos, block world.Block).
type PlayerBlockPlace struct {
	Entity ecs.Entity
	Pos    cube.Pos
	Block  world.Block
}

func (p PlayerBlockPlace) Player() ecs.Entity { return p.Entity }

// PlayerBlockPick is a cancellable event and corresponds to HandleBlockPick(ctx *player.Context, pos cube.Pos, block world.Block).
type PlayerBlockPick struct {
	Entity ecs.Entity
	Pos    cube.Pos
	Block  world.Block
}

func (p PlayerBlockPick) Player() ecs.Entity { return p.Entity }

// PlayerItemUse is a cancellable event and corresponds to HandleItemUse(ctx *player.Context).
type PlayerItemUse struct {
	Entity ecs.Entity
}

func (p PlayerItemUse) Player() ecs.Entity { return p.Entity }

// PlayerItemUseOnBlock is a cancellable event and corresponds to HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3).
type PlayerItemUseOnBlock struct {
	Entity   ecs.Entity
	Pos      cube.Pos
	Face     cube.Face
	ClickPos mgl64.Vec3
}

func (p PlayerItemUseOnBlock) Player() ecs.Entity { return p.Entity }

// PlayerItemUseOnEntity is a cancellable event and corresponds to HandleItemUseOnEntity(ctx *player.Context, target world.Entity).
type PlayerItemUseOnEntity struct {
	Entity ecs.Entity
	Target world.Entity
}

func (p PlayerItemUseOnEntity) Player() ecs.Entity { return p.Entity }

// PlayerItemRelease is a cancellable event and corresponds to HandleItemRelease(ctx *player.Context, item item.Stack, dur time.Duration).
type PlayerItemRelease struct {
	Entity ecs.Entity
	Item   item.Stack
	Dur    time.Duration
}

func (p PlayerItemRelease) Player() ecs.Entity { return p.Entity }

// PlayerItemConsume is a cancellable event and corresponds to HandleItemConsume(ctx *player.Context, item item.Stack).
type PlayerItemConsume struct {
	Entity ecs.Entity
	Item   item.Stack
}

func (p PlayerItemConsume) Player() ecs.Entity { return p.Entity }

// PlayerAttackEntity is a cancellable event and corresponds to HandleAttackEntity(ctx *player.Context, target world.Entity, force *float64, height *float64, critical *bool).
type PlayerAttackEntity struct {
	Entity   ecs.Entity
	Target   world.Entity
	Force    *float64
	Height   *float64
	Critical *bool
}

func (p PlayerAttackEntity) Player() ecs.Entity { return p.Entity }

// PlayerExperienceGain is a cancellable event and corresponds to HandleExperienceGain(ctx *player.Context, amount *int).
type PlayerExperienceGain struct {
	Entity ecs.Entity
	Amount *int
}

func (p PlayerExperienceGain) Player() ecs.Entity { return p.Entity }

// PlayerPunchAir is a cancellable event and corresponds to HandlePunchAir(ctx *player.Context).
type PlayerPunchAir struct {
	Entity ecs.Entity
}

func (p PlayerPunchAir) Player() ecs.Entity { return p.Entity }

// PlayerSignEdit is a cancellable event and corresponds to HandleSignEdit(ctx *player.Context, pos cube.Pos, frontSide bool, oldText string, newText string).
type PlayerSignEdit struct {
	Entity    ecs.Entity
	Pos       cube.Pos
	FrontSide bool
	OldText   string
	NewText   string
}

func (p PlayerSignEdit) Player() ecs.Entity { return p.Entity }

// PlayerLecternPageTurn is a cancellable event and corresponds to HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int).
type PlayerLecternPageTurn struct {
	Entity  ecs.Entity
	Pos     cube.Pos
	OldPage int
	NewPage *int
}

func (p PlayerLecternPageTurn) Player() ecs.Entity { return p.Entity }

// PlayerItemDamage is a cancellable event and corresponds to HandleItemDamage(ctx *player.Context, item item.Stack, damage int).
type PlayerItemDamage struct {
	Entity ecs.Entity
	Item   item.Stack
	Damage int
}

func (p PlayerItemDamage) Player() ecs.Entity { return p.Entity }

// PlayerItemPickup is a cancellable event and corresponds to HandleItemPickup(ctx *player.Context, item *item.Stack).
type PlayerItemPickup struct {
	Entity ecs.Entity
	Item   *item.Stack
}

func (p PlayerItemPickup) Player() ecs.Entity { return p.Entity }

// PlayerHeldSlotChange is a cancellable event and corresponds to HandleHeldSlotChange(ctx *player.Context, from int, to int).
type PlayerHeldSlotChange struct {
	Entity ecs.Entity
	From   int
	To     int
}

func (p PlayerHeldSlotChange) Player() ecs.Entity { return p.Entity }

// PlayerItemDrop is a cancellable event and corresponds to HandleItemDrop(ctx *player.Context, item item.Stack).
type PlayerItemDrop struct {
	Entity ecs.Entity
	Item   item.Stack
}

func (p PlayerItemDrop) Player() ecs.Entity { return p.Entity }

// PlayerTransfer is a cancellable event and corresponds to HandleTransfer(ctx *player.Context, addr *net.UDPAddr).
type PlayerTransfer struct {
	Entity ecs.Entity
	Addr   *net.UDPAddr
}

func (p PlayerTransfer) Player() ecs.Entity { return p.Entity }

// PlayerCommandExecution is a cancellable event and corresponds to HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string).
type PlayerCommandExecution struct {
	Entity  ecs.Entity
	Command cmd.Command
	Args    []string
}

func (p PlayerCommandExecution) Player() ecs.Entity { return p.Entity }

// PlayerJoin corresponds to HandleJoin(p *player.Player).
type PlayerJoin struct {
	Entity ecs.Entity
}

func (p PlayerJoin) Player() ecs.Entity { return p.Entity }

// PlayerQuit corresponds to HandleQuit(p *player.Player).
type PlayerQuit struct {
	Entity ecs.Entity
	wg     *sync.WaitGroup
}

func (p PlayerQuit) Player() ecs.Entity { return p.Entity }

// PlayerDiagnostics corresponds to HandleDiagnostics(p *player.Player, diagnostics session.Diagnostics).
type PlayerDiagnostics struct {
	Entity      ecs.Entity
	Diagnostics session.Diagnostics
}

func (p PlayerDiagnostics) Player() ecs.Entity { return p.Entity }

// PreQuit is special
type PlayerPreQuit struct {
	Entity ecs.Entity
}

func (p PlayerPreQuit) Player() ecs.Entity { return p.Entity }

// strictly internal, not for external consumption
type playerCreate struct {
	p *player.Player
}

type playerRemove struct {
	id ecs.Entity
	wg *sync.WaitGroup
}
