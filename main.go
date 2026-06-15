package main

import (
	"bytes"
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type ServerPlugin interface {
	Name() string
	Init(api *PluginAPI) error
}

type PluginAPI struct {
	Server *server.Server

	joinHandlers           []func(p *player.Player)
	playerHandlerFactories []func(p *player.Player) player.Handler
	inventoryFactories     []func(p *player.Player) inventory.Handler
	worldHandlerFactories  []func(w *world.World) world.Handler
}

func (api *PluginAPI) RegisterJoinHandler(f func(p *player.Player)) {
	api.joinHandlers = append(api.joinHandlers, f)
}

func (api *PluginAPI) RegisterPlayerHandler(f func(p *player.Player) player.Handler) {
	api.playerHandlerFactories = append(api.playerHandlerFactories, f)
}

func (api *PluginAPI) RegisterInventoryHandler(f func(p *player.Player) inventory.Handler) {
	api.inventoryFactories = append(api.inventoryFactories, f)
}

func (api *PluginAPI) RegisterWorldHandler(f func(w *world.World) world.Handler) {
	api.worldHandlerFactories = append(api.worldHandlerFactories, f)
}

func main() {
	chat.Global.Subscribe(chat.StdoutSubscriber{})

	plugins := openPlugins()

	config, err := server.DefaultConfig().Config(slog.Default())
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	configurePlugins(plugins, &config)

	beforeServerCreatePlugins(plugins, &config)
	srv := config.New()
	afterServerCreatePlugins(plugins, srv)
	srv.CloseOnProgramEnd()

	api := &PluginAPI{Server: srv}
	initPlugins(plugins, api)
	api.handleWorlds(srv.World(), srv.Nether(), srv.End())

	beforeServerListenPlugins(plugins, srv)
	srv.Listen()
	afterServerListenPlugins(plugins, srv)

	for p := range srv.Accept() {
		beforePlayerReadyPlugins(plugins, p)
		for _, f := range api.joinHandlers {
			f(p)
		}
		p.Handle(&PlayerMux{
			p:        p,
			handlers: api.playerHandlers(p),
		})
		api.handleInventories(p)
		afterPlayerReadyPlugins(plugins, p)
	}
}

func (api *PluginAPI) playerHandlers(p *player.Player) []player.Handler {
	handlers := make([]player.Handler, 0, len(api.playerHandlerFactories))
	for _, f := range api.playerHandlerFactories {
		if h := f(p); h != nil {
			handlers = append(handlers, h)
		}
	}
	return handlers
}

func (api *PluginAPI) inventoryHandlers(p *player.Player) []inventory.Handler {
	handlers := make([]inventory.Handler, 0, len(api.inventoryFactories))
	for _, f := range api.inventoryFactories {
		if h := f(p); h != nil {
			handlers = append(handlers, h)
		}
	}
	return handlers
}

func (api *PluginAPI) handleInventories(p *player.Player) {
	handlers := api.inventoryHandlers(p)
	if len(handlers) == 0 {
		return
	}

	p.Inventory().Handle(&InventoryMux{name: "inventory", handlers: handlers})
	p.Armour().Handle(&InventoryMux{name: "armour", handlers: handlers})
	p.EnderChestInventory().Handle(&InventoryMux{name: "ender_chest", handlers: handlers})
}

func (api *PluginAPI) handleWorlds(worlds ...*world.World) {
	for _, w := range worlds {
		if w == nil {
			continue
		}
		handlers := make([]world.Handler, 0, len(api.worldHandlerFactories))
		for _, f := range api.worldHandlerFactories {
			if h := f(w); h != nil {
				handlers = append(handlers, h)
			}
		}
		w.Handle(&WorldMux{w: w, handlers: handlers})
	}
}

type InventoryMux struct {
	inventory.NopHandler
	name     string
	handlers []inventory.Handler
}

func (m *InventoryMux) HandleTake(ctx *inventory.Context, slot int, it item.Stack) {
	for _, h := range m.handlers {
		h.HandleTake(ctx, slot, it)
	}
}
func (m *InventoryMux) HandlePlace(ctx *inventory.Context, slot int, it item.Stack) {
	for _, h := range m.handlers {
		h.HandlePlace(ctx, slot, it)
	}
}
func (m *InventoryMux) HandleDrop(ctx *inventory.Context, slot int, it item.Stack) {
	for _, h := range m.handlers {
		h.HandleDrop(ctx, slot, it)
	}
}

type PlayerMux struct {
	player.NopHandler
	p        *player.Player
	handlers []player.Handler
}

func (m *PlayerMux) HandleItemDrop(ctx *player.Context, s item.Stack) {
	for _, h := range m.handlers {
		h.HandleItemDrop(ctx, s)
	}
}
func (m *PlayerMux) HandleHeldSlotChange(ctx *player.Context, from, to int) {
	for _, h := range m.handlers {
		h.HandleHeldSlotChange(ctx, from, to)
	}
}
func (m *PlayerMux) HandleMove(ctx *player.Context, pos mgl64.Vec3, rot cube.Rotation) {
	for _, h := range m.handlers {
		h.HandleMove(ctx, pos, rot)
	}
}
func (m *PlayerMux) HandleJump(p *player.Player) {
	for _, h := range m.handlers {
		h.HandleJump(p)
	}
}
func (m *PlayerMux) HandleTeleport(ctx *player.Context, pos mgl64.Vec3) {
	for _, h := range m.handlers {
		h.HandleTeleport(ctx, pos)
	}
}
func (m *PlayerMux) HandleChangeWorld(p *player.Player, before, after *world.World) {
	for _, h := range m.handlers {
		h.HandleChangeWorld(p, before, after)
	}
}
func (m *PlayerMux) HandleToggleSprint(ctx *player.Context, after bool) {
	for _, h := range m.handlers {
		h.HandleToggleSprint(ctx, after)
	}
}
func (m *PlayerMux) HandleToggleSneak(ctx *player.Context, after bool) {
	for _, h := range m.handlers {
		h.HandleToggleSneak(ctx, after)
	}
}
func (m *PlayerMux) HandleCommandExecution(ctx *player.Context, command cmd.Command, args []string) {
	for _, h := range m.handlers {
		h.HandleCommandExecution(ctx, command, args)
	}
}
func (m *PlayerMux) HandleTransfer(ctx *player.Context, addr *net.UDPAddr) {
	for _, h := range m.handlers {
		h.HandleTransfer(ctx, addr)
	}
}
func (m *PlayerMux) HandleChat(ctx *player.Context, message *string) {
	for _, h := range m.handlers {
		h.HandleChat(ctx, message)
	}
}
func (m *PlayerMux) HandleSkinChange(ctx *player.Context, s *skin.Skin) {
	for _, h := range m.handlers {
		h.HandleSkinChange(ctx, s)
	}
}
func (m *PlayerMux) HandleFireExtinguish(ctx *player.Context, pos cube.Pos) {
	for _, h := range m.handlers {
		h.HandleFireExtinguish(ctx, pos)
	}
}
func (m *PlayerMux) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	for _, h := range m.handlers {
		h.HandleStartBreak(ctx, pos)
	}
}
func (m *PlayerMux) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	for _, h := range m.handlers {
		h.HandleBlockBreak(ctx, pos, drops, xp)
	}
}
func (m *PlayerMux) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block) {
	for _, h := range m.handlers {
		h.HandleBlockPlace(ctx, pos, b)
	}
}
func (m *PlayerMux) HandleBlockPick(ctx *player.Context, pos cube.Pos, b world.Block) {
	for _, h := range m.handlers {
		h.HandleBlockPick(ctx, pos, b)
	}
}
func (m *PlayerMux) HandleSignEdit(ctx *player.Context, pos cube.Pos, frontSide bool, oldText, newText string) {
	for _, h := range m.handlers {
		h.HandleSignEdit(ctx, pos, frontSide, oldText, newText)
	}
}
func (m *PlayerMux) HandleSleep(ctx *player.Context, sendReminder *bool) {
	for _, h := range m.handlers {
		h.HandleSleep(ctx, sendReminder)
	}
}
func (m *PlayerMux) HandleLecternPageTurn(ctx *player.Context, pos cube.Pos, oldPage int, newPage *int) {
	for _, h := range m.handlers {
		h.HandleLecternPageTurn(ctx, pos, oldPage, newPage)
	}
}
func (m *PlayerMux) HandleItemPickup(ctx *player.Context, s *item.Stack) {
	for _, h := range m.handlers {
		h.HandleItemPickup(ctx, s)
	}
}
func (m *PlayerMux) HandleItemUse(ctx *player.Context) {
	for _, h := range m.handlers {
		h.HandleItemUse(ctx)
	}
}
func (m *PlayerMux) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	for _, h := range m.handlers {
		h.HandleItemUseOnBlock(ctx, pos, face, clickPos)
	}
}
func (m *PlayerMux) HandleItemUseOnEntity(ctx *player.Context, e world.Entity) {
	for _, h := range m.handlers {
		h.HandleItemUseOnEntity(ctx, e)
	}
}
func (m *PlayerMux) HandleItemRelease(ctx *player.Context, s item.Stack, dur time.Duration) {
	for _, h := range m.handlers {
		h.HandleItemRelease(ctx, s, dur)
	}
}
func (m *PlayerMux) HandleItemConsume(ctx *player.Context, s item.Stack) {
	for _, h := range m.handlers {
		h.HandleItemConsume(ctx, s)
	}
}
func (m *PlayerMux) HandleItemDamage(ctx *player.Context, s item.Stack, damage *int) {
	for _, h := range m.handlers {
		h.HandleItemDamage(ctx, s, damage)
	}
}
func (m *PlayerMux) HandleAttackEntity(ctx *player.Context, e world.Entity, force, height *float64, critical *bool) {
	for _, h := range m.handlers {
		h.HandleAttackEntity(ctx, e, force, height, critical)
	}
}
func (m *PlayerMux) HandleExperienceGain(ctx *player.Context, amount *int) {
	for _, h := range m.handlers {
		h.HandleExperienceGain(ctx, amount)
	}
}
func (m *PlayerMux) HandlePunchAir(ctx *player.Context) {
	for _, h := range m.handlers {
		h.HandlePunchAir(ctx)
	}
}
func (m *PlayerMux) HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource) {
	for _, h := range m.handlers {
		h.HandleHurt(ctx, damage, immune, attackImmunity, src)
	}
}
func (m *PlayerMux) HandleHeal(ctx *player.Context, health *float64, src world.HealingSource) {
	for _, h := range m.handlers {
		h.HandleHeal(ctx, health, src)
	}
}
func (m *PlayerMux) HandleFoodLoss(ctx *player.Context, from int, to *int) {
	for _, h := range m.handlers {
		h.HandleFoodLoss(ctx, from, to)
	}
}
func (m *PlayerMux) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	for _, h := range m.handlers {
		h.HandleDeath(p, src, keepInv)
	}
}
func (m *PlayerMux) HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World) {
	for _, h := range m.handlers {
		h.HandleRespawn(p, pos, w)
	}
}
func (m *PlayerMux) HandleQuit(p *player.Player) {
	for _, h := range m.handlers {
		h.HandleQuit(p)
	}
}
func (m *PlayerMux) HandleDiagnostics(p *player.Player, d session.Diagnostics) {
	for _, h := range m.handlers {
		h.HandleDiagnostics(p, d)
	}
}

type WorldMux struct {
	world.NopHandler
	w        *world.World
	handlers []world.Handler
}

func (m *WorldMux) HandleLiquidFlow(ctx *world.Context, from, into cube.Pos, liquid world.Liquid, replaced world.Block) {
	for _, h := range m.handlers {
		h.HandleLiquidFlow(ctx, from, into, liquid, replaced)
	}
}
func (m *WorldMux) HandleLiquidDecay(ctx *world.Context, pos cube.Pos, before, after world.Liquid) {
	for _, h := range m.handlers {
		h.HandleLiquidDecay(ctx, pos, before, after)
	}
}
func (m *WorldMux) HandleLiquidHarden(ctx *world.Context, pos cube.Pos, liquidHardened, otherLiquid, newBlock world.Block) {
	for _, h := range m.handlers {
		h.HandleLiquidHarden(ctx, pos, liquidHardened, otherLiquid, newBlock)
	}
}
func (m *WorldMux) HandleSound(ctx *world.Context, s world.Sound, pos mgl64.Vec3) {
	for _, h := range m.handlers {
		h.HandleSound(ctx, s, pos)
	}
}
func (m *WorldMux) HandleFireSpread(ctx *world.Context, from, to cube.Pos) {
	for _, h := range m.handlers {
		h.HandleFireSpread(ctx, from, to)
	}
}
func (m *WorldMux) HandleBlockBurn(ctx *world.Context, pos cube.Pos) {
	for _, h := range m.handlers {
		h.HandleBlockBurn(ctx, pos)
	}
}
func (m *WorldMux) HandleCropTrample(ctx *world.Context, pos cube.Pos) {
	for _, h := range m.handlers {
		h.HandleCropTrample(ctx, pos)
	}
}
func (m *WorldMux) HandleLeavesDecay(ctx *world.Context, pos cube.Pos) {
	for _, h := range m.handlers {
		h.HandleLeavesDecay(ctx, pos)
	}
}
func (m *WorldMux) HandleEntitySpawn(tx *world.Tx, e world.Entity) {
	for _, h := range m.handlers {
		h.HandleEntitySpawn(tx, e)
	}
}
func (m *WorldMux) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	for _, h := range m.handlers {
		h.HandleEntityDespawn(tx, e)
	}
}
func (m *WorldMux) HandleExplosion(ctx *world.Context, pos mgl64.Vec3, entities *[]world.Entity, blocks *[]cube.Pos, itemDropChance *float64, spawnFire *bool) {
	for _, h := range m.handlers {
		h.HandleExplosion(ctx, pos, entities, blocks, itemDropChance, spawnFire)
	}
}
func (m *WorldMux) HandleClose(tx *world.Tx) {
	for _, h := range m.handlers {
		h.HandleClose(tx)
	}
}

type loadedPlugin struct {
	file string
	p    *plugin.Plugin
}

func openPlugins() []loadedPlugin {
	pluginDir := "./plugins"

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		log.Fatalf("failed to create plugins directory: %v", err)
	}

	if runtime.GOOS == "windows" {
		log.Printf("Go plugins are not supported on windows/%s. Run this server on Linux or WSL to load plugins from %s.", runtime.GOARCH, pluginDir)
		return nil
	}

	files, err := filepath.Glob(filepath.Join(pluginDir, "*.so"))
	if err != nil {
		log.Fatalf("failed to scan plugins: %v", err)
	}

	plugins := make([]loadedPlugin, 0, len(files))
	for _, file := range files {
		if p := openPlugin(file); p != nil {
			plugins = append(plugins, loadedPlugin{file: file, p: p})
		}
	}
	return plugins
}

func openPlugin(file string) *plugin.Plugin {
	log.Printf("loading plugin %s...", file)

	if invalid, reason := invalidPluginFile(file); invalid {
		log.Printf("skipping plugin %s: %s", file, reason)
		return nil
	}

	p, err := plugin.Open(file)
	if err != nil {
		log.Printf("failed to open plugin %s: %v", file, err)
		return nil
	}
	return p
}

func configurePlugins(plugins []loadedPlugin, config *server.Config) {
	for _, pl := range plugins {
		if err := configurePluginObject(pl.p, pl.file, config); err != nil {
			log.Printf("ConfigureServer hook failed in %s: %v", pl.file, err)
		}
		configureFunctionHooks(pl.p, pl.file, config)
	}
}

func beforeServerCreatePlugins(plugins []loadedPlugin, config *server.Config) {
	for _, pl := range plugins {
		runConfigObjectHook(pl.p, pl.file, config, "BeforeServerCreate")
		runConfigFunctionHook(pl.p, pl.file, config, "BeforeServerCreate")
	}
}

func afterServerCreatePlugins(plugins []loadedPlugin, srv *server.Server) {
	for _, pl := range plugins {
		runServerObjectHook(pl.p, pl.file, srv, "AfterServerCreate", "OnServerCreated")
		runServerFunctionHook(pl.p, pl.file, srv, "AfterServerCreate", "OnServerCreated")
	}
}

func beforeServerListenPlugins(plugins []loadedPlugin, srv *server.Server) {
	for _, pl := range plugins {
		runServerObjectHook(pl.p, pl.file, srv, "BeforeServerListen")
		runServerFunctionHook(pl.p, pl.file, srv, "BeforeServerListen")
	}
}

func afterServerListenPlugins(plugins []loadedPlugin, srv *server.Server) {
	for _, pl := range plugins {
		runServerObjectHook(pl.p, pl.file, srv, "AfterServerListen")
		runServerFunctionHook(pl.p, pl.file, srv, "AfterServerListen")
	}
}

func beforePlayerReadyPlugins(plugins []loadedPlugin, p *player.Player) {
	for _, pl := range plugins {
		runPlayerObjectHook(pl.p, pl.file, p, "BeforePlayerReady")
		runPlayerFunctionHook(pl.p, pl.file, p, "BeforePlayerReady")
	}
}

func afterPlayerReadyPlugins(plugins []loadedPlugin, p *player.Player) {
	for _, pl := range plugins {
		runPlayerObjectHook(pl.p, pl.file, p, "AfterPlayerReady")
		runPlayerFunctionHook(pl.p, pl.file, p, "AfterPlayerReady")
	}
}

func initPlugins(plugins []loadedPlugin, api *PluginAPI) {
	for _, pl := range plugins {
		if err := initPluginObject(pl.p, pl.file, api); err != nil {
			log.Printf("failed to initialise plugin object %s: %v", pl.file, err)
		}
		registerFunctionHooks(pl.p, pl.file, api)
	}
}

func invalidPluginFile(file string) (bool, string) {
	data, err := os.ReadFile(file)
	if err != nil {
		return true, "failed to read file before opening: " + err.Error()
	}
	if bytes.Contains(data, []byte("-buildmode=c-shared")) {
		return true, "built with -buildmode=c-shared; rebuild it with -buildmode=plugin"
	}
	return false, ""
}

func configurePluginObject(p *plugin.Plugin, file string, config *server.Config) error {
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil
	}

	if ptr, ok := sym.(*ServerPlugin); ok {
		return configurePluginValue(*ptr, config)
	}
	return configurePluginValue(sym, config)
}

func configurePluginValue(value any, config *server.Config) error {
	switch h := value.(type) {
	case interface{ ConfigureServer(*server.Config) error }:
		return h.ConfigureServer(config)
	case interface{ ConfigureServer(*server.Config) }:
		h.ConfigureServer(config)
	case interface{ Configure(*server.Config) error }:
		return h.Configure(config)
	case interface{ Configure(*server.Config) }:
		h.Configure(config)
	}
	return nil
}

func initPluginObject(p *plugin.Plugin, file string, api *PluginAPI) error {
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil
	}

	if ptr, ok := sym.(*ServerPlugin); ok {
		if err := (*ptr).Init(api); err != nil {
			return err
		}
		registerObjectHooks(*ptr, api)
		log.Printf("loaded plugin %s from %s", (*ptr).Name(), file)
		return nil
	}

	pl, ok := sym.(ServerPlugin)
	if !ok {
		log.Printf("%s has Plugin symbol, but it does not implement ServerPlugin", file)
		return nil
	}
	if err := pl.Init(api); err != nil {
		return err
	}
	registerObjectHooks(pl, api)
	log.Printf("loaded plugin %s from %s", pl.Name(), file)
	return nil
}

func registerObjectHooks(pl ServerPlugin, api *PluginAPI) {
	if h, ok := pl.(interface{ OnPlayerJoin(*player.Player) }); ok {
		api.RegisterJoinHandler(h.OnPlayerJoin)
	}
	if h, ok := pl.(interface {
		PlayerHandler(*player.Player) player.Handler
	}); ok {
		api.RegisterPlayerHandler(h.PlayerHandler)
	}
	if h, ok := pl.(interface {
		InventoryHandler(*player.Player) inventory.Handler
	}); ok {
		api.RegisterInventoryHandler(h.InventoryHandler)
	}
	if h, ok := pl.(interface {
		WorldHandler(*world.World) world.Handler
	}); ok {
		api.RegisterWorldHandler(h.WorldHandler)
	}
	if h, ok := pl.(interface {
		OnEntitySpawn(*world.Tx, world.Entity)
	}); ok {
		api.RegisterWorldHandler(func(*world.World) world.Handler {
			return &WorldEventHandler{spawn: h.OnEntitySpawn}
		})
	}
	if h, ok := pl.(interface {
		OnEntityDespawn(*world.Tx, world.Entity)
	}); ok {
		api.RegisterWorldHandler(func(*world.World) world.Handler {
			return &WorldEventHandler{despawn: h.OnEntityDespawn}
		})
	}
}

func configureFunctionHooks(p *plugin.Plugin, file string, config *server.Config) {
	if sym, err := p.Lookup("ConfigureServer"); err == nil {
		switch f := sym.(type) {
		case func(*server.Config) error:
			if err := f(config); err != nil {
				log.Printf("ConfigureServer hook failed in %s: %v", file, err)
			}
		case func(*server.Config):
			f(config)
		default:
			log.Printf("ConfigureServer hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("Configure"); err == nil {
		switch f := sym.(type) {
		case func(*server.Config) error:
			if err := f(config); err != nil {
				log.Printf("Configure hook failed in %s: %v", file, err)
			}
		case func(*server.Config):
			f(config)
		default:
			log.Printf("Configure hook in %s has an unsupported signature", file)
		}
	}
}

func runConfigObjectHook(p *plugin.Plugin, file string, config *server.Config, method string) {
	value, ok := pluginObject(p)
	if !ok {
		return
	}
	if err := callConfigHook(value, config, method); err != nil {
		log.Printf("%s hook failed in %s: %v", method, file, err)
	}
}

func runConfigFunctionHook(p *plugin.Plugin, file string, config *server.Config, name string) {
	sym, err := p.Lookup(name)
	if err != nil {
		return
	}
	if err := callConfigHook(sym, config, name); err != nil {
		log.Printf("%s hook failed in %s: %v", name, file, err)
	}
}

func callConfigHook(value any, config *server.Config, name string) error {
	switch name {
	case "BeforeServerCreate":
		switch h := value.(type) {
		case interface{ BeforeServerCreate(*server.Config) error }:
			return h.BeforeServerCreate(config)
		case interface{ BeforeServerCreate(*server.Config) }:
			h.BeforeServerCreate(config)
		case func(*server.Config) error:
			return h(config)
		case func(*server.Config):
			h(config)
		default:
			return nil
		}
	}
	return nil
}

func runServerObjectHook(p *plugin.Plugin, file string, srv *server.Server, names ...string) {
	value, ok := pluginObject(p)
	if !ok {
		return
	}
	for _, name := range names {
		if err := callServerHook(value, srv, name); err != nil {
			log.Printf("%s hook failed in %s: %v", name, file, err)
		}
	}
}

func runServerFunctionHook(p *plugin.Plugin, file string, srv *server.Server, names ...string) {
	for _, name := range names {
		sym, err := p.Lookup(name)
		if err != nil {
			continue
		}
		if err := callServerHook(sym, srv, name); err != nil {
			log.Printf("%s hook failed in %s: %v", name, file, err)
		}
	}
}

func callServerHook(value any, srv *server.Server, name string) error {
	switch name {
	case "AfterServerCreate":
		switch h := value.(type) {
		case interface{ AfterServerCreate(*server.Server) error }:
			return h.AfterServerCreate(srv)
		case interface{ AfterServerCreate(*server.Server) }:
			h.AfterServerCreate(srv)
		}
	case "OnServerCreated":
		switch h := value.(type) {
		case interface{ OnServerCreated(*server.Server) error }:
			return h.OnServerCreated(srv)
		case interface{ OnServerCreated(*server.Server) }:
			h.OnServerCreated(srv)
		}
	case "BeforeServerListen":
		switch h := value.(type) {
		case interface{ BeforeServerListen(*server.Server) error }:
			return h.BeforeServerListen(srv)
		case interface{ BeforeServerListen(*server.Server) }:
			h.BeforeServerListen(srv)
		}
	case "AfterServerListen":
		switch h := value.(type) {
		case interface{ AfterServerListen(*server.Server) error }:
			return h.AfterServerListen(srv)
		case interface{ AfterServerListen(*server.Server) }:
			h.AfterServerListen(srv)
		}
	}
	switch h := value.(type) {
	case func(*server.Server) error:
		return h(srv)
	case func(*server.Server):
		h(srv)
	}
	return nil
}

func runPlayerObjectHook(p *plugin.Plugin, file string, player *player.Player, name string) {
	value, ok := pluginObject(p)
	if !ok {
		return
	}
	if err := callPlayerHook(value, player, name); err != nil {
		log.Printf("%s hook failed in %s: %v", name, file, err)
	}
}

func runPlayerFunctionHook(p *plugin.Plugin, file string, player *player.Player, name string) {
	sym, err := p.Lookup(name)
	if err != nil {
		return
	}
	if err := callPlayerHook(sym, player, name); err != nil {
		log.Printf("%s hook failed in %s: %v", name, file, err)
	}
}

func callPlayerHook(value any, p *player.Player, name string) error {
	switch name {
	case "BeforePlayerReady":
		switch h := value.(type) {
		case interface{ BeforePlayerReady(*player.Player) error }:
			return h.BeforePlayerReady(p)
		case interface{ BeforePlayerReady(*player.Player) }:
			h.BeforePlayerReady(p)
		case func(*player.Player) error:
			return h(p)
		case func(*player.Player):
			h(p)
		}
	case "AfterPlayerReady":
		switch h := value.(type) {
		case interface{ AfterPlayerReady(*player.Player) error }:
			return h.AfterPlayerReady(p)
		case interface{ AfterPlayerReady(*player.Player) }:
			h.AfterPlayerReady(p)
		case func(*player.Player) error:
			return h(p)
		case func(*player.Player):
			h(p)
		}
	}
	return nil
}

func pluginObject(p *plugin.Plugin) (any, bool) {
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, false
	}
	if ptr, ok := sym.(*ServerPlugin); ok {
		return *ptr, true
	}
	return sym, true
}

func registerFunctionHooks(p *plugin.Plugin, file string, api *PluginAPI) {
	if sym, err := p.Lookup("Init"); err == nil {
		switch f := sym.(type) {
		case func(*server.Server) error:
			if err := f(api.Server); err != nil {
				log.Printf("Init hook failed in %s: %v", file, err)
			}
		case func(*PluginAPI) error:
			if err := f(api); err != nil {
				log.Printf("Init hook failed in %s: %v", file, err)
			}
		case func(*server.Server):
			f(api.Server)
		case func(*PluginAPI):
			f(api)
		default:
			log.Printf("Init hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("OnPlayerJoin"); err == nil {
		if f, ok := sym.(func(*player.Player)); ok {
			api.RegisterJoinHandler(f)
		} else {
			log.Printf("OnPlayerJoin hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("PlayerHandler"); err == nil {
		if f, ok := sym.(func(*player.Player) player.Handler); ok {
			api.RegisterPlayerHandler(f)
		} else {
			log.Printf("PlayerHandler hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("InventoryHandler"); err == nil {
		if f, ok := sym.(func(*player.Player) inventory.Handler); ok {
			api.RegisterInventoryHandler(f)
		} else {
			log.Printf("InventoryHandler hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("WorldHandler"); err == nil {
		if f, ok := sym.(func(*world.World) world.Handler); ok {
			api.RegisterWorldHandler(f)
		} else {
			log.Printf("WorldHandler hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("OnEntitySpawn"); err == nil {
		if f, ok := sym.(func(*world.Tx, world.Entity)); ok {
			api.RegisterWorldHandler(func(*world.World) world.Handler {
				return &WorldEventHandler{spawn: f}
			})
		} else {
			log.Printf("OnEntitySpawn hook in %s has an unsupported signature", file)
		}
	}

	if sym, err := p.Lookup("OnEntityDespawn"); err == nil {
		if f, ok := sym.(func(*world.Tx, world.Entity)); ok {
			api.RegisterWorldHandler(func(*world.World) world.Handler {
				return &WorldEventHandler{despawn: f}
			})
		} else {
			log.Printf("OnEntityDespawn hook in %s has an unsupported signature", file)
		}
	}
}

type WorldEventHandler struct {
	world.NopHandler
	spawn   func(*world.Tx, world.Entity)
	despawn func(*world.Tx, world.Entity)
}

func (h *WorldEventHandler) HandleEntitySpawn(tx *world.Tx, e world.Entity) {
	if h.spawn != nil {
		h.spawn(tx, e)
	}
}

func (h *WorldEventHandler) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	if h.despawn != nil {
		h.despawn(tx, e)
	}
}
