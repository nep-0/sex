# sex (Simple EXpose)

`sex` is a simple debug server that exposes host resources over HTTP with configuration driven by TOML. Each route can specify an exposed path, expose type, resource type, custom headers, and whether to watch for changes.

## Features

- Configurable HTTP server address and timeouts
- Per-route expose type: `http`, `sse`, or `websocket`
- Resource types: `file` or `image`
- Custom headers (e.g., `Content-Type`)
- Optional file change watching for SSE/websocket streaming (fsnotify)
- Embedded web UI for subscribing to streams

## Install

Download a release binary from GitHub Releases, or install with Go:

```bash
go install github.com/nep-0/sex/cmd/sex@latest
```

## Build

```bash
go build ./cmd/sex
```

## Run

```bash
./sex --config sex.toml
```

Open the UI at:

```
http://localhost:8080/
```

## Example config

```toml
[server]
address = ":8080"
read_timeout = "5s"
write_timeout = "10s"

[[route]]
path = "/logs"
expose_type = "http"
resource_type = "file"
source = "/var/log/system.log"
watch = false
headers = { "Content-Type" = "text/plain" }

[[route]]
path = "/events"
expose_type = "sse"
resource_type = "file"
source = "/tmp/app.log"
watch = true
timeout = "2m"

[[route]]
path = "/ws"
expose_type = "websocket"
resource_type = "file"
source = "/tmp/app.log"
watch = true
timeout = "2m"

[[route]]
path = "/ws-image"
expose_type = "websocket"
resource_type = "image"
source = "/tmp/frame.jpg"
watch = true
timeout = "2m"
```

## Route fields

- `path`: HTTP path to expose
- `expose_type`: `http`, `sse`, or `websocket` (recommended)
- `resource_type`: `file` or `image`
- `source`: filesystem path to serve
- `headers`: map of extra response headers
- `watch`: when `true`, SSE/websocket streams updates on file changes
- `timeout`: connection timeout (default `30s`)

## Notes

- SSE/websocket `watch=true` uses fsnotify on the parent directory of the source file.
- Image SSE emits data URLs; image WebSocket sends binary JPEG frames.
