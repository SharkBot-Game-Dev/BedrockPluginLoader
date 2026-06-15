# exampleplugin

This plugin demonstrates the function-hook API exposed by BedrockPluginLoader.

Implemented hooks:

- `Init(*server.Server)`
- `OnPlayerJoin(*player.Player)`
- `PlayerHandler(*player.Player) player.Handler`
- `WorldHandler(*world.World) world.Handler`

Build it from the repository root on a Go platform that supports plugins, such
as Linux:

```sh
go build -buildmode=plugin -o plugins/exampleplugin.so ./exampleplugin
```

Note: Go does not support `-buildmode=plugin` on `windows/amd64`, so build the
`.so` on a supported platform and place it in `plugins/`.

Then start the server normally. The loader scans `plugins/*.so` and registers
the exported hooks automatically.

Try typing `cancel` in chat to see an event cancelled, or any other message to
see it rewritten with an `[example]` prefix.
