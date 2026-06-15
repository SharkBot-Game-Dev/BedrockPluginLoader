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
