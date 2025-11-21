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
)

// Player events.
// These map 1:1 to github.com/df-mc/dragonfly/server/player.Handler methods.

type PlayerEvent interface {
	PlayerRef() *Player
}

// PlayerMove is a cancellable event and corresponds to HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation).
type PlayerMove struct {
	Player *Player
	NewPos mgl64.Vec3
	NewRot cube.Rotation
}

func (p PlayerMove) PlayerRef() *Player { return p.Player }

// PlayerJump corresponds to HandleJump(p *player.Player).
type PlayerJump struct {
	Player *Player
}

func (p PlayerJump) PlayerRef() *Player { return p.Player }

// PlayerTeleport is a cancellable event and corresponds to HandleTeleport(ctx *player.Context, pos mgl64.Vec3).
type PlayerTeleport struct {
	Player *Player
	Pos    mgl64.Vec3
}

func (p PlayerTeleport) PlayerRef() *Player { return p.Player }

// PlayerChangeWorld corresponds to HandleChangeWorld(p *player.Player, before *world.World, after *world.World).
type PlayerChangeWorld struct {
	Player *Player
	Before *world.World
	After  *world.World
}

func (p PlayerChangeWorld) PlayerRef() *Player { return p.Player }

// PlayerToggleSprint is a cancellable event and corresponds to HandleToggleSprint(ctx *player.Context, after bool).
type PlayerToggleSprint struct {
	Player *Player
	After  bool
}

func (p PlayerToggleSprint) PlayerRef() *Player { return p.Player }

// PlayerToggleSneak is a cancellable event and corresponds to HandleToggleSneak(ctx *player.Context, after bool).
type PlayerToggleSneak struct {
	Player *Player
	After  bool
}

func (p PlayerToggleSneak) PlayerRef() *Player { return p.Player }

// PlayerChat is a cancellable event and corresponds to HandleChat(ctx *player.Context, message *string).
type PlayerChat struct {
	Player  *Player
	Message *string
}

func (p PlayerChat) PlayerRef() *Player { return p.Player }

// PlayerFoodLoss is a cancellable event and corresponds to HandleFoodLoss(ctx *player.Context, from int, to *int).
type PlayerFoodLoss struct {
	Player *Player
	From   int
	To     *int
}

func (p PlayerFoodLoss) PlayerRef() *Player { return p.Player }

// PlayerHeal is a cancellable event and corresponds to HandleHeal(ctx *player.Context, health *float64, src world.HealingSource).
type PlayerHeal struct {
	Player *Player
	Health *float64
	Src    world.HealingSource
}

func (p PlayerHeal) PlayerRef() *Player { return p.Player }

// PlayerHurt is a cancellable event and corresponds to HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource).
type PlayerHurt struct {
	Player         *Player
	Damage         *float64
	Immune         bool
	AttackImmunity *time.Duration
	Src            world.DamageSource
}

func (p PlayerHurt) PlayerRef() *Player { return p.Player }

// PlayerDeath corresponds to HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool).
type PlayerDeath struct {
	Player  *Player
	Src     world.DamageSource
	KeepInv *bool
}

func (p PlayerDeath) PlayerRef() *Player { return p.Player }

// PlayerRespawn corresponds to HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World).
type PlayerRespawn struct {
	Player *Player
	Pos    *mgl64.Vec3
	W      **world.World
}

func (p PlayerRespawn) PlayerRef() *Player { return p.Player }

// PlayerSkinChange is a cancellable event and corresponds to HandleSkinChange(ctx *player.Context, skin *skin.Skin).
type PlayerSkinChange struct {
	Player *Player
	Skin   *skin.Skin
}

func (p PlayerSkinChange) PlayerRef() *Player { return p.Player }

// PlayerFireExtinguish is a cancellable event and corresponds to HandleFireExtinguish(ctx *player.Context, pos cube.Pos).
type PlayerFireExtinguish struct {
	Player *Player
	Pos    cube.Pos
}

func (p PlayerFireExtinguish) PlayerRef() *Player { return p.Player }

// PlayerStartBreak is a cancellable event and corresponds to HandleStartBreak(ctx *player.Context, pos cube.Pos).
type PlayerStartBreak struct {
	Player *Player
	Pos    cube.Pos
}

func (p PlayerStartBreak) PlayerRef() *Player { return p.Player }

// PlayerBlockBreak is a cancellable event and corresponds to HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int).
type PlayerBlockBreak struct {
	Player *Player
	Pos    cube.Pos
	Drops  *[]item.Stack
	Xp     *int
}

func (p PlayerBlockBreak) PlayerRef() *Player { return p.Player }

// PlayerBlockPlace is a cancellable event and corresponds to HandleBlockPlace(ctx *player.Context, pos cube.Pos, block world.Block).
type PlayerBlockPlace struct {
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

func (p PlayerBlockPlace) PlayerRef() *Player { return p.Player }

// PlayerBlockPick is a cancellable event and corresponds to HandleBlockPick(ctx *player.Context, pos cube.Pos, block world.Block).
type PlayerBlockPick struct {
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

func (p PlayerBlockPick) PlayerRef() *Player { return p.Player }

// PlayerItemUse is a cancellable event and corresponds to HandleItemUse(ctx *player.Context).
type PlayerItemUse struct {
	Player *Player
}

func (p PlayerItemUse) PlayerRef() *Player { return p.Player }

// PlayerItemUseOnBlock is a cancellable event and corresponds to HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3).
type PlayerItemUseOnBlock struct {
	Player   *Player
	Pos      cube.Pos
	Face     cube.Face
	ClickPos mgl64.Vec3
}

func (p PlayerItemUseOnBlock) PlayerRef() *Player { return p.Player }

// PlayerItemUseOnEntity is a cancellable event and corresponds to HandleItemUseOnEntity(ctx *player.Context, target world.Entity).
type PlayerItemUseOnEntity struct {
	Player *Player
	Target world.Entity
}

func (p PlayerItemUseOnEntity) PlayerRef() *Player { return p.Player }

// PlayerItemRelease is a cancellable event and corresponds to HandleItemRelease(ctx *player.Context, item item.Stack, dur time.Duration).
type PlayerItemRelease struct {
	Player *Player
	Item   item.Stack
	Dur    time.Duration
}

func (p PlayerItemRelease) PlayerRef() *Player { return p.Player }

// PlayerItemConsume is a cancellable event and corresponds to HandleItemConsume(ctx *player.Context, item item.Stack).
type PlayerItemConsume struct {
	Player *Player
	Item   item.Stack
}

func (p PlayerItemConsume) PlayerRef() *Player { return p.Player }

// PlayerAttackEntity is a cancellable event and corresponds to HandleAttackEntity(ctx *player.Context, target world.Entity, force *float64, height *float64, critical *bool).
type PlayerAttackEntity struct {
	Player   *Player
	Target   world.Entity
	Force    *float64
	Height   *float64
	Critical *bool
}

func (p PlayerAttackEntity) PlayerRef() *Player { return p.Player }

// PlayerExperienceGain is a cancellable event and corresponds to HandleExperienceGain(ctx *player.Context, amount *int).
type PlayerExperienceGain struct {
	Player *Player
	Amount *int
}

func (p PlayerExperienceGain) PlayerRef() *Player { return p.Player }

// PlayerPunchAir is a cancellable event and corresponds to HandlePunchAir(ctx *player.Context).
type PlayerPunchAir struct {
	Player *Player
}

func (p PlayerPunchAir) PlayerRef() *Player { return p.Player }

// PlayerSignEdit is a cancellable event and corresponds to HandleSignEdit(ctx *player.Context, pos cube.Pos, frontSide bool, oldText string, newText string).
type PlayerSignEdit struct {
	Player    *Player
	Pos       cube.Pos
	FrontSide bool
	OldText   string
	NewText   string
}

func (p PlayerSignEdit) PlayerRef() *Player { return p.Player }

// PlayerLecternPageTurn is a cancellable event and corresponds to HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int).
type PlayerLecternPageTurn struct {
	Player  *Player
	Pos     cube.Pos
	OldPage int
	NewPage *int
}

func (p PlayerLecternPageTurn) PlayerRef() *Player { return p.Player }

// PlayerItemDamage is a cancellable event and corresponds to HandleItemDamage(ctx *player.Context, item item.Stack, damage int).
type PlayerItemDamage struct {
	Player *Player
	Item   item.Stack
	Damage int
}

func (p PlayerItemDamage) PlayerRef() *Player { return p.Player }

// PlayerItemPickup is a cancellable event and corresponds to HandleItemPickup(ctx *player.Context, item *item.Stack).
type PlayerItemPickup struct {
	Player *Player
	Item   *item.Stack
}

func (p PlayerItemPickup) PlayerRef() *Player { return p.Player }

// PlayerHeldSlotChange is a cancellable event and corresponds to HandleHeldSlotChange(ctx *player.Context, from int, to int).
type PlayerHeldSlotChange struct {
	Player *Player
	From   int
	To     int
}

func (p PlayerHeldSlotChange) PlayerRef() *Player { return p.Player }

// PlayerItemDrop is a cancellable event and corresponds to HandleItemDrop(ctx *player.Context, item item.Stack).
type PlayerItemDrop struct {
	Player *Player
	Item   item.Stack
}

func (p PlayerItemDrop) PlayerRef() *Player { return p.Player }

// PlayerTransfer is a cancellable event and corresponds to HandleTransfer(ctx *player.Context, addr *net.UDPAddr).
type PlayerTransfer struct {
	Player *Player
	Addr   *net.UDPAddr
}

func (p PlayerTransfer) PlayerRef() *Player { return p.Player }

// PlayerCommandExecution is a cancellable event and corresponds to HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string).
type PlayerCommandExecution struct {
	Player  *Player
	Command cmd.Command
	Args    []string
}

func (p PlayerCommandExecution) PlayerRef() *Player { return p.Player }

// PlayerJoin corresponds to HandleJoin(p *player.Player).
type PlayerJoin struct {
	Player *Player
}

func (p PlayerJoin) PlayerRef() *Player { return p.Player }

// PlayerQuit corresponds to HandleQuit(p *player.Player).
type PlayerQuit struct {
	Player *Player
	wg     *sync.WaitGroup
}

func (p PlayerQuit) PlayerRef() *Player { return p.Player }

// PlayerDiagnostics corresponds to HandleDiagnostics(p *player.Player, diagnostics session.Diagnostics).
type PlayerDiagnostics struct {
	Player      *Player
	Diagnostics session.Diagnostics
}

func (p PlayerDiagnostics) PlayerRef() *Player { return p.Player }

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
