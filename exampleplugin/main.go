package main

import (
	"log"
	"strings"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func Init(srv *server.Server) {
	log.Printf("[exampleplugin] loaded on %s", srv.World().Dimension())
}

func OnPlayerJoin(p *player.Player) {
	log.Printf("[exampleplugin] %s joined", p.Name())
	p.Message("exampleplugin is enabled.")
}

func PlayerHandler(p *player.Player) player.Handler {
	return &examplePlayerHandler{p: p}
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

func (h *examplePlayerHandler) HandleQuit(p *player.Player) {
	log.Printf("[exampleplugin] %s quit", p.Name())
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

func main() {}
