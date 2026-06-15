# moderatorplugin

Moderation plugin for BedrockPluginLoader that blocks URL spam in chat.

Protection rules:

- Blocks messages containing more than one URL.
- Blocks the same player from sending more than two URL messages within two minutes.
- Blocks the same player from repeating the same URL within one minute.

Build it from the repository root on a Go platform that supports plugins, such as Linux:

```sh
go build -buildmode=plugin -o plugins/moderatorplugin.so ./moderatorplugin
```

Note: Go does not support `-buildmode=plugin` on `windows/amd64`, so build the `.so` on a supported platform and place it in `plugins/`.
