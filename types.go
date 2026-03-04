package sex

type ExposeType string

const (
	ExposeHTTP      ExposeType = "http"
	ExposeSSE       ExposeType = "sse"
	ExposeWebSocket ExposeType = "websocket"
)

type ResourceType string

const (
	ResourceFile  ResourceType = "file"
	ResourceImage ResourceType = "image"
)
