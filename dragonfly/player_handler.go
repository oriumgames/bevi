package dragonfly

import (
	"context"
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
	"github.com/oriumgames/ark/ecs"
	"github.com/oriumgames/bevi"
)

// playerHandler bridges Dragonfly player events to the ECS.
type playerHandler struct {
	ctx    context.Context
	srv    *Server
	world  *ecs.World
	mapper *ecs.Map1[Player]

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

	// internal
	create bevi.EventWriter[playerCreate]
	remove bevi.EventWriter[playerRemove]
}

func newPlayerHandler(ctx context.Context, app *bevi.App, srv *Server) *playerHandler {
	return &playerHandler{
		ctx:    ctx,
		srv:    srv,
		world:  app.World(),
		mapper: ecs.NewMap1[Player](app.World()),

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

		// internal
		create: bevi.WriterFor[playerCreate](app.Events()),
		remove: bevi.WriterFor[playerRemove](app.Events()),
	}
}

func (h *playerHandler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.move.EmitResult(PlayerMove{
		Player: ip,
		NewPos: newPos,
		NewRot: newRot,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleJump(p *player.Player) {
	ip, ok := h.srv.Player(p.UUID())
	if !ok {
		return
	}
	h.jump.Emit(PlayerJump{
		Player: ip,
	})
}

func (h *playerHandler) HandleTeleport(ctx *player.Context, pos mgl64.Vec3) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.teleport.EmitResult(PlayerTeleport{
		Player: ip,
		Pos:    pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleChangeWorld(p *player.Player, before, after *world.World) {
	ip, ok := h.srv.Player(p.UUID())
	if !ok {
		return
	}
	h.changeWorld.Emit(PlayerChangeWorld{
		Player: ip,
		Before: before,
		After:  after,
	})
}

func (h *playerHandler) HandleToggleSprint(ctx *player.Context, after bool) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.toggleSprint.EmitResult(PlayerToggleSprint{
		Player: ip,
		After:  after,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleToggleSneak(ctx *player.Context, after bool) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.toggleSneak.EmitResult(PlayerToggleSneak{
		Player: ip,
		After:  after,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleChat(ctx *player.Context, message *string) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.chat.EmitResult(PlayerChat{
		Player:  ip,
		Message: message,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleFoodLoss(ctx *player.Context, from int, to *int) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.foodLoss.EmitResult(PlayerFoodLoss{
		Player: ip,
		From:   from,
		To:     to,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHeal(ctx *player.Context, health *float64, src world.HealingSource) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.heal.EmitResult(PlayerHeal{
		Player: ip,
		Health: health,
		Src:    src,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.hurt.EmitResult(PlayerHurt{
		Player:         ip,
		Damage:         damage,
		Immune:         immune,
		AttackImmunity: attackImmunity,
		Src:            src,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	ip, ok := h.srv.Player(p.UUID())
	if !ok {
		return
	}
	h.death.Emit(PlayerDeath{
		Player:  ip,
		Src:     src,
		KeepInv: keepInv,
	})
}

func (h *playerHandler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World) {
	ip, ok := h.srv.Player(p.UUID())
	if !ok {
		return
	}
	h.respawn.Emit(PlayerRespawn{
		Player: ip,
		Pos:    pos,
		W:      w,
	})
}

func (h *playerHandler) HandleSkinChange(ctx *player.Context, skin *skin.Skin) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.skinChange.EmitResult(PlayerSkinChange{
		Player: ip,
		Skin:   skin,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleFireExtinguish(ctx *player.Context, pos cube.Pos) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.fireExtinguish.EmitResult(PlayerFireExtinguish{
		Player: ip,
		Pos:    pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.startBreak.EmitResult(PlayerStartBreak{
		Player: ip,
		Pos:    pos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockBreak.EmitResult(PlayerBlockBreak{
		Player: ip,
		Pos:    pos,
		Drops:  drops,
		XP:     xp,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockPlace.EmitResult(PlayerBlockPlace{
		Player: ip,
		Pos:    pos,
		Block:  b,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleBlockPick(ctx *player.Context, pos cube.Pos, b world.Block) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.blockPick.EmitResult(PlayerBlockPick{
		Player: ip,
		Pos:    pos,
		Block:  b,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUse(ctx *player.Context) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUse.EmitResult(PlayerItemUse{
		Player: ip,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUseOnBlock.EmitResult(PlayerItemUseOnBlock{
		Player:   ip,
		Pos:      pos,
		Face:     face,
		ClickPos: clickPos,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemUseOnEntity.EmitResult(PlayerItemUseOnEntity{
		Player: ip,
		Target: e,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemRelease(ctx *player.Context, item item.Stack, dur time.Duration) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemRelease.EmitResult(PlayerItemRelease{
		Player: ip,
		Item:   item,
		Dur:    dur,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemConsume(ctx *player.Context, item item.Stack) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemConsume.EmitResult(PlayerItemConsume{
		Player: ip,
		Item:   item,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleAttackEntity(ctx *player.Context, e world.Entity, force, height *float64, critical *bool) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.attackEntity.EmitResult(PlayerAttackEntity{
		Player:   ip,
		Target:   e,
		Force:    force,
		Height:   height,
		Critical: critical,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleExperienceGain(ctx *player.Context, amount *int) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.experienceGain.EmitResult(PlayerExperienceGain{
		Player: ip,
		Amount: amount,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandlePunchAir(ctx *player.Context) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.punchAir.EmitResult(PlayerPunchAir{
		Player: ip,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleSignEdit(ctx *player.Context, pos cube.Pos, front bool, oldText, newText string) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.signEdit.EmitResult(PlayerSignEdit{
		Player:    ip,
		Pos:       pos,
		FrontSide: front,
		OldText:   oldText,
		NewText:   newText,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.lecternPageTurn.EmitResult(PlayerLecternPageTurn{
		Player:  ip,
		Pos:     pos,
		OldPage: oldPage,
		NewPage: newPage,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemDamage(ctx *player.Context, i item.Stack, damage int) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemDamage.EmitResult(PlayerItemDamage{
		Player: ip,
		Item:   i,
		Damage: damage,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemPickup(ctx *player.Context, i *item.Stack) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemPickup.EmitResult(PlayerItemPickup{
		Player: ip,
		Item:   i,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleHeldSlotChange(ctx *player.Context, from, to int) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.heldSlotChange.EmitResult(PlayerHeldSlotChange{
		Player: ip,
		From:   from,
		To:     to,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleItemDrop(ctx *player.Context, it item.Stack) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.itemDrop.EmitResult(PlayerItemDrop{
		Player: ip,
		Item:   it,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleTransfer(ctx *player.Context, addr *net.UDPAddr) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.transfer.EmitResult(PlayerTransfer{
		Player: ip,
		Addr:   addr,
	}).WaitCancelled(h.ctx) {
		ctx.Cancel()
	}
}

func (h *playerHandler) HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string) {
	ip, ok := h.srv.Player(ctx.Val().UUID())
	if !ok {
		return
	}
	if h.commandExecution.EmitResult(PlayerCommandExecution{
		Player:  ip,
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
	h.remove.Emit(playerRemove{
		id: p.UUID(),
	})
}

func (h *playerHandler) HandleDiagnostics(p *player.Player, d session.Diagnostics) {
	ip, ok := h.srv.Player(p.UUID())
	if !ok {
		return
	}
	h.diagnostics.Emit(PlayerDiagnostics{
		Player:      ip,
		Diagnostics: d,
	})
}
