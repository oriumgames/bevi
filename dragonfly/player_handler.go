package dragonfly

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
	ctx   context.Context
	srv   *Server
	world *bevi.World

	keepInv atomic.Bool

	move             bevi.EventWriter[PlayerMove]
	jump             bevi.EventWriter[PlayerJump]
	teleport         bevi.EventWriter[PlayerTeleport]
	changeWorld      bevi.EventWriter[PlayerChangeWorld]
	toggleSprint     bevi.EventWriter[PlayerToggleSprint]
	toggleSneak      bevi.EventWriter[PlayerToggleSneak]
	chat             bevi.EventWriter[PlayerChat]
	foodLoss         bevi.EventWriter[PlayerFoodLoss]
	heal             bevi.EventWriter[PlayerHeal]
	hurt             bevi.EventWriter[PlayerHurt]
	death            bevi.EventWriter[PlayerDeath]
	respawn          bevi.EventWriter[PlayerRespawn]
	skinChange       bevi.EventWriter[PlayerSkinChange]
	fireExtinguish   bevi.EventWriter[PlayerFireExtinguish]
	startBreak       bevi.EventWriter[PlayerStartBreak]
	blockBreak       bevi.EventWriter[PlayerBlockBreak]
	blockPlace       bevi.EventWriter[PlayerBlockPlace]
	blockPick        bevi.EventWriter[PlayerBlockPick]
	itemUse          bevi.EventWriter[PlayerItemUse]
	itemUseOnBlock   bevi.EventWriter[PlayerItemUseOnBlock]
	itemUseOnEntity  bevi.EventWriter[PlayerItemUseOnEntity]
	itemRelease      bevi.EventWriter[PlayerItemRelease]
	itemConsume      bevi.EventWriter[PlayerItemConsume]
	attackEntity     bevi.EventWriter[PlayerAttackEntity]
	experienceGain   bevi.EventWriter[PlayerExperienceGain]
	punchAir         bevi.EventWriter[PlayerPunchAir]
	signEdit         bevi.EventWriter[PlayerSignEdit]
	lecternPageTurn  bevi.EventWriter[PlayerLecternPageTurn]
	itemDamage       bevi.EventWriter[PlayerItemDamage]
	itemPickup       bevi.EventWriter[PlayerItemPickup]
	heldSlotChange   bevi.EventWriter[PlayerHeldSlotChange]
	itemDrop         bevi.EventWriter[PlayerItemDrop]
	transfer         bevi.EventWriter[PlayerTransfer]
	commandExecution bevi.EventWriter[PlayerCommandExecution]
	join             bevi.EventWriter[PlayerJoin]
	quit             bevi.EventWriter[PlayerQuit]
	diagnostics      bevi.EventWriter[PlayerDiagnostics]
	preQuit          bevi.EventWriter[PlayerPreQuit]

	// internal
	create bevi.EventWriter[playerCreate]
	remove bevi.EventWriter[playerRemove]
}

func newPlayerHandler(ctx context.Context, app *bevi.App, srv *Server) *playerHandler {
	return &playerHandler{
		ctx:   ctx,
		srv:   srv,
		world: app.World(),

		move:             bevi.WriterFor[PlayerMove](app.Events()),
		jump:             bevi.WriterFor[PlayerJump](app.Events()),
		teleport:         bevi.WriterFor[PlayerTeleport](app.Events()),
		changeWorld:      bevi.WriterFor[PlayerChangeWorld](app.Events()),
		toggleSprint:     bevi.WriterFor[PlayerToggleSprint](app.Events()),
		toggleSneak:      bevi.WriterFor[PlayerToggleSneak](app.Events()),
		chat:             bevi.WriterFor[PlayerChat](app.Events()),
		foodLoss:         bevi.WriterFor[PlayerFoodLoss](app.Events()),
		heal:             bevi.WriterFor[PlayerHeal](app.Events()),
		hurt:             bevi.WriterFor[PlayerHurt](app.Events()),
		death:            bevi.WriterFor[PlayerDeath](app.Events()),
		respawn:          bevi.WriterFor[PlayerRespawn](app.Events()),
		skinChange:       bevi.WriterFor[PlayerSkinChange](app.Events()),
		fireExtinguish:   bevi.WriterFor[PlayerFireExtinguish](app.Events()),
		startBreak:       bevi.WriterFor[PlayerStartBreak](app.Events()),
		blockBreak:       bevi.WriterFor[PlayerBlockBreak](app.Events()),
		blockPlace:       bevi.WriterFor[PlayerBlockPlace](app.Events()),
		blockPick:        bevi.WriterFor[PlayerBlockPick](app.Events()),
		itemUse:          bevi.WriterFor[PlayerItemUse](app.Events()),
		itemUseOnBlock:   bevi.WriterFor[PlayerItemUseOnBlock](app.Events()),
		itemUseOnEntity:  bevi.WriterFor[PlayerItemUseOnEntity](app.Events()),
		itemRelease:      bevi.WriterFor[PlayerItemRelease](app.Events()),
		itemConsume:      bevi.WriterFor[PlayerItemConsume](app.Events()),
		attackEntity:     bevi.WriterFor[PlayerAttackEntity](app.Events()),
		experienceGain:   bevi.WriterFor[PlayerExperienceGain](app.Events()),
		punchAir:         bevi.WriterFor[PlayerPunchAir](app.Events()),
		signEdit:         bevi.WriterFor[PlayerSignEdit](app.Events()),
		lecternPageTurn:  bevi.WriterFor[PlayerLecternPageTurn](app.Events()),
		itemDamage:       bevi.WriterFor[PlayerItemDamage](app.Events()),
		itemPickup:       bevi.WriterFor[PlayerItemPickup](app.Events()),
		heldSlotChange:   bevi.WriterFor[PlayerHeldSlotChange](app.Events()),
		itemDrop:         bevi.WriterFor[PlayerItemDrop](app.Events()),
		transfer:         bevi.WriterFor[PlayerTransfer](app.Events()),
		commandExecution: bevi.WriterFor[PlayerCommandExecution](app.Events()),
		join:             bevi.WriterFor[PlayerJoin](app.Events()),
		quit:             bevi.WriterFor[PlayerQuit](app.Events()),
		diagnostics:      bevi.WriterFor[PlayerDiagnostics](app.Events()),
		preQuit:          bevi.WriterFor[PlayerPreQuit](app.Events()),

		// internal
		create: bevi.WriterFor[playerCreate](app.Events()),
		remove: bevi.WriterFor[playerRemove](app.Events()),
	}
}

func (h *playerHandler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.move.EmitResult(PlayerMove{
		Player: dp,
		NewPos: newPos,
		NewRot: newRot,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleJump(p *player.Player) {
	dp, ok := h.srv.PlayerByUUID(p.UUID())
	if !ok {
		return
	}
	h.jump.Emit(PlayerJump{
		Player: dp,
	})
}

func (h *playerHandler) HandleTeleport(ctx *player.Context, pos mgl64.Vec3) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.teleport.EmitResult(PlayerTeleport{
		Player: dp,
		Pos:    pos,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleChangeWorld(p *player.Player, before *world.World, after *world.World) {
	dp, ok := h.srv.PlayerByUUID(p.UUID())
	if !ok {
		return
	}
	h.changeWorld.Emit(PlayerChangeWorld{
		Player: dp,
		Before: before,
		After:  after,
	})
}

func (h *playerHandler) HandleToggleSprint(ctx *player.Context, after bool) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.toggleSprint.EmitResult(PlayerToggleSprint{
		Player: dp,
		After:  after,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleToggleSneak(ctx *player.Context, after bool) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.toggleSneak.EmitResult(PlayerToggleSneak{
		Player: dp,
		After:  after,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleChat(ctx *player.Context, message *string) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.chat.EmitResult(PlayerChat{
		Player:  dp,
		Message: message,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleFoodLoss(ctx *player.Context, from int, to *int) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.foodLoss.EmitResult(PlayerFoodLoss{
		Player: dp,
		From:   from,
		To:     to,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHeal(ctx *player.Context, health *float64, src world.HealingSource) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.heal.EmitResult(PlayerHeal{
		Player: dp,
		Health: health,
		Src:    src,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource) {
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
}

func (h *playerHandler) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	// Custom Death override
	*keepInv = h.keepInv.Load()
}

func (h *playerHandler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World) {
	dp, ok := h.srv.PlayerByUUID(p.UUID())
	if !ok {
		return
	}
	h.respawn.Emit(PlayerRespawn{
		Player: dp,
		Pos:    pos,
		W:      w,
	})
}

func (h *playerHandler) HandleSkinChange(ctx *player.Context, skin *skin.Skin) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.skinChange.EmitResult(PlayerSkinChange{
		Player: dp,
		Skin:   skin,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleFireExtinguish(ctx *player.Context, pos cube.Pos) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.fireExtinguish.EmitResult(PlayerFireExtinguish{
		Player: dp,
		Pos:    pos,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.startBreak.EmitResult(PlayerStartBreak{
		Player: dp,
		Pos:    pos,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockBreak.EmitResult(PlayerBlockBreak{
		Player: dp,
		Pos:    pos,
		Drops:  drops,
		Xp:     xp,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, block world.Block) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockPlace.EmitResult(PlayerBlockPlace{
		Player: dp,
		Pos:    pos,
		Block:  block,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockPick(ctx *player.Context, pos cube.Pos, block world.Block) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockPick.EmitResult(PlayerBlockPick{
		Player: dp,
		Pos:    pos,
		Block:  block,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUse(ctx *player.Context) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUse.EmitResult(PlayerItemUse{
		Player: dp,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUseOnBlock.EmitResult(PlayerItemUseOnBlock{
		Player:   dp,
		Pos:      pos,
		Face:     face,
		ClickPos: clickPos,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUseOnEntity(ctx *player.Context, target world.Entity) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUseOnEntity.EmitResult(PlayerItemUseOnEntity{
		Player: dp,
		Target: target,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemRelease(ctx *player.Context, item item.Stack, dur time.Duration) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemRelease.EmitResult(PlayerItemRelease{
		Player: dp,
		Item:   item,
		Dur:    dur,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemConsume(ctx *player.Context, item item.Stack) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemConsume.EmitResult(PlayerItemConsume{
		Player: dp,
		Item:   item,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleAttackEntity(ctx *player.Context, target world.Entity, force *float64, height *float64, critical *bool) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.attackEntity.EmitResult(PlayerAttackEntity{
		Player:   dp,
		Target:   target,
		Force:    force,
		Height:   height,
		Critical: critical,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleExperienceGain(ctx *player.Context, amount *int) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.experienceGain.EmitResult(PlayerExperienceGain{
		Player: dp,
		Amount: amount,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandlePunchAir(ctx *player.Context) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.punchAir.EmitResult(PlayerPunchAir{
		Player: dp,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleSignEdit(ctx *player.Context, pos cube.Pos, frontSide bool, oldText string, newText string) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.signEdit.EmitResult(PlayerSignEdit{
		Player:    dp,
		Pos:       pos,
		FrontSide: frontSide,
		OldText:   oldText,
		NewText:   newText,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.lecternPageTurn.EmitResult(PlayerLecternPageTurn{
		Player:  dp,
		Pos:     pos,
		OldPage: oldPage,
		NewPage: newPage,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemDamage(ctx *player.Context, item item.Stack, damage int) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemDamage.EmitResult(PlayerItemDamage{
		Player: dp,
		Item:   item,
		Damage: damage,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemPickup(ctx *player.Context, item *item.Stack) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemPickup.EmitResult(PlayerItemPickup{
		Player: dp,
		Item:   item,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHeldSlotChange(ctx *player.Context, from int, to int) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.heldSlotChange.EmitResult(PlayerHeldSlotChange{
		Player: dp,
		From:   from,
		To:     to,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemDrop(ctx *player.Context, item item.Stack) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemDrop.EmitResult(PlayerItemDrop{
		Player: dp,
		Item:   item,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleTransfer(ctx *player.Context, addr *net.UDPAddr) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.transfer.EmitResult(PlayerTransfer{
		Player: dp,
		Addr:   addr,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string) {
	dp, ok := h.srv.PlayerByUUID(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.commandExecution.EmitResult(PlayerCommandExecution{
		Player:  dp,
		Command: command,
		Args:    args,
	}).Wait(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleJoin(p *player.Player) {
	h.create.Emit(playerCreate{
		p: p,
	})
}

func (h *playerHandler) HandleQuit(p *player.Player) {
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
}

func (h *playerHandler) HandleDiagnostics(p *player.Player, diagnostics session.Diagnostics) {
	dp, ok := h.srv.PlayerByUUID(p.UUID())
	if !ok {
		return
	}
	h.diagnostics.Emit(PlayerDiagnostics{
		Player:      dp,
		Diagnostics: diagnostics,
	})
}
