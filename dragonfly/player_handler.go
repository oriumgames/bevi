package dragonfly

import (
	"context"
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
	"github.com/oriumgames/bevi"
)

// playerHandler bridges Dragonfly player events to the ECS.
type playerHandler struct {
	ctx   context.Context
	srv   *Server
	world *ecs.World

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
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.move.EmitResult(PlayerMove{
		Entity: id,
		NewPos: newPos,
		NewRot: newRot,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleJump(p *player.Player) {
	id, ok := h.srv.PlayerEntity(p.UUID())
	if !ok {
		return
	}
	h.jump.Emit(PlayerJump{
		Entity: id,
	})
}

func (h *playerHandler) HandleTeleport(ctx *player.Context, pos mgl64.Vec3) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.teleport.EmitResult(PlayerTeleport{
		Entity: id,
		Pos:    pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleChangeWorld(p *player.Player, before *world.World, after *world.World) {
	id, ok := h.srv.PlayerEntity(p.UUID())
	if !ok {
		return
	}
	h.changeWorld.Emit(PlayerChangeWorld{
		Entity: id,
		Before: before,
		After:  after,
	})
}

func (h *playerHandler) HandleToggleSprint(ctx *player.Context, after bool) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.toggleSprint.EmitResult(PlayerToggleSprint{
		Entity: id,
		After:  after,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleToggleSneak(ctx *player.Context, after bool) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.toggleSneak.EmitResult(PlayerToggleSneak{
		Entity: id,
		After:  after,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleChat(ctx *player.Context, message *string) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.chat.EmitResult(PlayerChat{
		Entity:  id,
		Message: message,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleFoodLoss(ctx *player.Context, from int, to *int) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.foodLoss.EmitResult(PlayerFoodLoss{
		Entity: id,
		From:   from,
		To:     to,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHeal(ctx *player.Context, health *float64, src world.HealingSource) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.heal.EmitResult(PlayerHeal{
		Entity: id,
		Health: health,
		Src:    src,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.hurt.EmitResult(PlayerHurt{
		Entity:         id,
		Damage:         damage,
		Immune:         immune,
		AttackImmunity: attackImmunity,
		Src:            src,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	id, ok := h.srv.PlayerEntity(p.UUID())
	if !ok {
		return
	}
	h.death.Emit(PlayerDeath{
		Entity:  id,
		Src:     src,
		KeepInv: keepInv,
	})
}

func (h *playerHandler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World) {
	id, ok := h.srv.PlayerEntity(p.UUID())
	if !ok {
		return
	}
	h.respawn.Emit(PlayerRespawn{
		Entity: id,
		Pos:    pos,
		W:      w,
	})
}

func (h *playerHandler) HandleSkinChange(ctx *player.Context, skin *skin.Skin) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.skinChange.EmitResult(PlayerSkinChange{
		Entity: id,
		Skin:   skin,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleFireExtinguish(ctx *player.Context, pos cube.Pos) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.fireExtinguish.EmitResult(PlayerFireExtinguish{
		Entity: id,
		Pos:    pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.startBreak.EmitResult(PlayerStartBreak{
		Entity: id,
		Pos:    pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockBreak.EmitResult(PlayerBlockBreak{
		Entity: id,
		Pos:    pos,
		Drops:  drops,
		Xp:     xp,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, block world.Block) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockPlace.EmitResult(PlayerBlockPlace{
		Entity: id,
		Pos:    pos,
		Block:  block,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockPick(ctx *player.Context, pos cube.Pos, block world.Block) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockPick.EmitResult(PlayerBlockPick{
		Entity: id,
		Pos:    pos,
		Block:  block,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUse(ctx *player.Context) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUse.EmitResult(PlayerItemUse{
		Entity: id,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUseOnBlock.EmitResult(PlayerItemUseOnBlock{
		Entity:   id,
		Pos:      pos,
		Face:     face,
		ClickPos: clickPos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUseOnEntity(ctx *player.Context, target world.Entity) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUseOnEntity.EmitResult(PlayerItemUseOnEntity{
		Entity: id,
		Target: target,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemRelease(ctx *player.Context, item item.Stack, dur time.Duration) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemRelease.EmitResult(PlayerItemRelease{
		Entity: id,
		Item:   item,
		Dur:    dur,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemConsume(ctx *player.Context, item item.Stack) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemConsume.EmitResult(PlayerItemConsume{
		Entity: id,
		Item:   item,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleAttackEntity(ctx *player.Context, target world.Entity, force *float64, height *float64, critical *bool) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.attackEntity.EmitResult(PlayerAttackEntity{
		Entity:   id,
		Target:   target,
		Force:    force,
		Height:   height,
		Critical: critical,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleExperienceGain(ctx *player.Context, amount *int) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.experienceGain.EmitResult(PlayerExperienceGain{
		Entity: id,
		Amount: amount,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandlePunchAir(ctx *player.Context) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.punchAir.EmitResult(PlayerPunchAir{
		Entity: id,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleSignEdit(ctx *player.Context, pos cube.Pos, frontSide bool, oldText string, newText string) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.signEdit.EmitResult(PlayerSignEdit{
		Entity:    id,
		Pos:       pos,
		FrontSide: frontSide,
		OldText:   oldText,
		NewText:   newText,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.lecternPageTurn.EmitResult(PlayerLecternPageTurn{
		Entity:  id,
		Pos:     pos,
		OldPage: oldPage,
		NewPage: newPage,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemDamage(ctx *player.Context, item item.Stack, damage int) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemDamage.EmitResult(PlayerItemDamage{
		Entity: id,
		Item:   item,
		Damage: damage,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemPickup(ctx *player.Context, item *item.Stack) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemPickup.EmitResult(PlayerItemPickup{
		Entity: id,
		Item:   item,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHeldSlotChange(ctx *player.Context, from int, to int) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.heldSlotChange.EmitResult(PlayerHeldSlotChange{
		Entity: id,
		From:   from,
		To:     to,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemDrop(ctx *player.Context, item item.Stack) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemDrop.EmitResult(PlayerItemDrop{
		Entity: id,
		Item:   item,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleTransfer(ctx *player.Context, addr *net.UDPAddr) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.transfer.EmitResult(PlayerTransfer{
		Entity: id,
		Addr:   addr,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string) {
	id, ok := h.srv.PlayerEntity(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.commandExecution.EmitResult(PlayerCommandExecution{
		Entity:  id,
		Command: command,
		Args:    args,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleJoin(p *player.Player) {
	h.create.Emit(playerCreate{
		p: p,
	})
}

func (h *playerHandler) HandleQuit(p *player.Player) {
	id, ok := h.srv.PlayerEntity(p.UUID())
	if !ok {
		return
	}

	h.preQuit.Emit(PlayerPreQuit{
		Entity: id,
	})

	var wg sync.WaitGroup
	wg.Add(1)

	h.remove.Emit(playerRemove{
		id: id,
		wg: &wg,
	})

	wg.Wait()
}

func (h *playerHandler) HandleDiagnostics(p *player.Player, diagnostics session.Diagnostics) {
	id, ok := h.srv.PlayerEntity(p.UUID())
	if !ok {
		return
	}
	h.diagnostics.Emit(PlayerDiagnostics{
		Entity:      id,
		Diagnostics: diagnostics,
	})
}
