package main

import (
	"log"
	"strings"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func ConfigureServer(config *server.Config) {
	config.Name = "BedrockPluginLoader Example"
	config.AuthDisabled = true
	config.MaxPlayers = 20
	config.MaxChunkRadius = 16
	config.ReadOnlyWorld = false
}

func BeforeServerCreate(config *server.Config) {
	log.Printf("[exampleplugin] before server create: name=%q maxPlayers=%d", config.Name, config.MaxPlayers)
}

func AfterServerCreate(srv *server.Server) {
	log.Printf("[exampleplugin] after server create: worlds are ready")
}

func BeforeServerListen(srv *server.Server) {
	log.Printf("[exampleplugin] before server listen")
}

func AfterServerListen(srv *server.Server) {
	log.Printf("[exampleplugin] after server listen")
}

func Init(srv *server.Server) {
	log.Printf("[exampleplugin] loaded on %s", srv.World().Dimension())
}

func BeforePlayerReady(p *player.Player) {
	log.Printf("[exampleplugin] before player ready: %s", p.Name())
}

func OnPlayerJoin(p *player.Player) {
	log.Printf("[exampleplugin] %s joined", p.Name())
	p.Message("exampleplugin is enabled.")
}

func AfterPlayerReady(p *player.Player) {
	log.Printf("[exampleplugin] after player ready: %s", p.Name())
}

func PlayerHandler(p *player.Player) player.Handler {
	return &examplePlayerHandler{p: p}
}

func InventoryHandler(p *player.Player) inventory.Handler {
	return &exampleInventoryHandler{p: p}
}

func WorldHandler(w *world.World) world.Handler {
	return &exampleWorldHandler{}
}

type examplePlayerHandler struct {
	player.NopHandler
	p *player.Player
}

func (h *examplePlayerHandler) HandleChat(ctx *player.Context, message *string) {
	if strings.EqualFold(*message, "cancel") {
		h.p.Message("Your message was cancelled by exampleplugin.")
		ctx.Cancel()
		return
	}
	*message = "[example] " + *message
}

func (h *examplePlayerHandler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	log.Printf("[exampleplugin] %s broke block at %v with %d drops and %d xp", h.p.Name(), pos, len(*drops), *xp)
}

func (h *examplePlayerHandler) HandleItemUse(ctx *player.Context) {
	mainHand, _ := h.p.HeldItems()
	log.Printf("[exampleplugin] %s used %v", h.p.Name(), mainHand)
}

func (h *examplePlayerHandler) HandleAttackEntity(ctx *player.Context, target world.Entity, force, height *float64, critical *bool) {
	if living, ok := target.(entity.Living); ok {
		log.Printf("[exampleplugin] %s attacked living entity %s with %.1f health", h.p.Name(), target.H().Type().EncodeEntity(), living.Health())
		return
	}
	log.Printf("[exampleplugin] %s attacked entity %s", h.p.Name(), target.H().Type().EncodeEntity())
}

func (h *examplePlayerHandler) HandleQuit(p *player.Player) {
	log.Printf("[exampleplugin] %s quit", p.Name())
}

type exampleInventoryHandler struct {
	inventory.NopHandler
	p *player.Player
}

func (h *exampleInventoryHandler) HandleTake(ctx *inventory.Context, slot int, it item.Stack) {
	log.Printf("[exampleplugin] %s took %v from inventory slot %d", h.p.Name(), it, slot)
}

func (h *exampleInventoryHandler) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	log.Printf("[exampleplugin] %s placed %v into inventory slot %d", h.p.Name(), it, slot)
}

func (h *exampleInventoryHandler) HandleDrop(ctx *inventory.Context, slot int, it item.Stack) {
	log.Printf("[exampleplugin] %s dropped %v from inventory slot %d", h.p.Name(), it, slot)
}

type exampleWorldHandler struct {
	world.NopHandler
}

func (h *exampleWorldHandler) HandleSound(ctx *world.Context, sound world.Sound, pos mgl64.Vec3) {
	log.Printf("[exampleplugin] sound %T at %v", sound, pos)
}

func (h *exampleWorldHandler) HandleExplosion(ctx *world.Context, pos mgl64.Vec3, entities *[]world.Entity, blocks *[]cube.Pos, itemDropChance *float64, spawnFire *bool) {
	log.Printf("[exampleplugin] explosion at %v affecting %d entities and %d blocks", pos, len(*entities), len(*blocks))
}

func (h *exampleWorldHandler) HandleEntitySpawn(tx *world.Tx, e world.Entity) {
	if living, ok := e.(entity.Living); ok {
		log.Printf("[exampleplugin] living entity spawned: %s with %.1f health", e.H().Type().EncodeEntity(), living.Health())
		return
	}
	log.Printf("[exampleplugin] entity spawned: %s", e.H().Type().EncodeEntity())
}

func (h *exampleWorldHandler) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	log.Printf("[exampleplugin] entity despawned: %s", e.H().Type().EncodeEntity())
}

func main() {}
