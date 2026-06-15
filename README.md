# BedrockPluginLoader

Dragonfly-based Minecraft Bedrock server plugin loader.

## Platform note

This loader uses Go's standard `plugin` package, so runtime plugin loading only
works on platforms supported by `-buildmode=plugin`, such as Linux.

Windows currently prints `plugin: not implemented` when `plugin.Open` is used.
Run the server on Linux or WSL if you want to load `plugins/*.so`.

Build plugins with `-buildmode=plugin`, not `-buildmode=c-shared`:

```sh
go build -buildmode=plugin -o plugins/example.so ./exampleplugin
```

The server binary and plugin should be built with the same Go toolchain, module
path, and dependency versions.

## Server configuration hooks

Plugins may export `ConfigureServer(*server.Config)` or
`ConfigureServer(*server.Config) error`. This hook runs before the Dragonfly
server is created, so plugins can change detailed server behaviour:

```go
func ConfigureServer(config *server.Config) {
	config.Name = "Custom Server"
	config.AuthDisabled = true
	config.MaxPlayers = 20
	config.MaxChunkRadius = 16
}
```

Anything exposed on `server.Config` can be changed here, including status
providers, join allowers, world/player providers, generators, resources,
authentication, player limits, world save behaviour, and resource-pack settings.

## Lifecycle hooks

Plugins may export lifecycle hooks to insert behaviour around the Dragonfly
calls controlled by this loader:

- `BeforeServerCreate(*server.Config)`
- `AfterServerCreate(*server.Server)` or `OnServerCreated(*server.Server)`
- `BeforeServerListen(*server.Server)`
- `AfterServerListen(*server.Server)`
- `BeforePlayerReady(*player.Player)`
- `AfterPlayerReady(*player.Player)`

Each hook may also return an `error`.

Go cannot replace arbitrary already-compiled Dragonfly functions at runtime. To
override internals that Dragonfly does not expose through `server.Config`,
handlers, providers, registries, or generators, fork Dragonfly, add explicit hook
points around the target function or variable, and use a `replace
github.com/df-mc/dragonfly => ./path/to/fork` directive.
