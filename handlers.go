package sex

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"
)

func buildRouteHandler(route ParsedRoute) (http.HandlerFunc, error) {
	expose := strings.ToLower(route.ExposeType)
	resource := strings.ToLower(route.ResourceType)

	switch expose {
	case string(ExposeHTTP):
		switch resource {
		case string(ResourceFile), string(ResourceImage):
			return handleFileHTTP(route), nil
		default:
			return nil, ErrUnsupportedType
		}
	case string(ExposeSSE):
		switch resource {
		case string(ResourceFile):
			return handleFileSSE(route), nil
		case string(ResourceImage):
			return handleImageSSE(route), nil
		default:
			return nil, ErrUnsupportedType
		}
	case string(ExposeWebSocket):
		switch resource {
		case string(ResourceFile):
			return handleFileWebSocket(route), nil
		case string(ResourceImage):
			return handleImageWebSocket(route), nil
		default:
			return nil, ErrUnsupportedType
		}
	default:
		return nil, ErrUnsupportedType
	}
}

func handleFileHTTP(route ParsedRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ApplyHeaders(w, route.Headers)
		info, err := os.Stat(route.Source)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "source not found", http.StatusNotFound)
				return
			}
			http.Error(w, "stat source failed", http.StatusInternalServerError)
			return
		}
		if info.IsDir() {
			http.Error(w, "source is a directory", http.StatusBadRequest)
			return
		}

		f, err := os.Open(route.Source)
		if err != nil {
			http.Error(w, "open source failed", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		if _, ok := route.Headers["Content-Type"]; !ok {
			contentType := mime.TypeByExtension(filepath.Ext(route.Source))
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	}
}

func handleFileSSE(route ParsedRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ApplyHeaders(w, route.Headers)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
		defer cancel()

		initial, err := readFile(route.Source)
		if err != nil {
			http.Error(w, "read source failed", http.StatusInternalServerError)
			return
		}
		writeSSE(w, "snapshot", string(initial))
		flusher.Flush()

		if !route.Watch {
			return
		}

		watcher, err := newFSWatcher(route.Source)
		if err != nil {
			http.Error(w, "watch source failed", http.StatusInternalServerError)
			return
		}
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events():
				if !ok {
					return
				}
				if event.Err != nil {
					writeSSE(w, "error", event.Err.Error())
					flusher.Flush()
					return
				}
				writeSSE(w, "update", string(event.Data))
				flusher.Flush()
			}
		}
	}
}

func handleImageSSE(route ParsedRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ApplyHeaders(w, route.Headers)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
		defer cancel()

		initial, err := os.ReadFile(route.Source)
		if err != nil {
			http.Error(w, "read source failed", http.StatusInternalServerError)
			return
		}
		writeSSE(w, "snapshot", encodeBase64(initial))
		flusher.Flush()

		if !route.Watch {
			return
		}

		watcher, err := newFSWatcher(route.Source)
		if err != nil {
			http.Error(w, "watch source failed", http.StatusInternalServerError)
			return
		}
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events():
				if !ok {
					return
				}
				if event.Err != nil {
					writeSSE(w, "error", event.Err.Error())
					flusher.Flush()
					return
				}
				writeSSE(w, "update", encodeBase64(event.Data))
				flusher.Flush()
			}
		}
	}
}

func handleFileWebSocket(route ParsedRoute) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ApplyHeaders(w, route.Headers)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
		defer cancel()

		initial, err := readFile(route.Source)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("read source failed"))
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, initial)

		if !route.Watch {
			<-ctx.Done()
			return
		}

		watcher, err := newFSWatcher(route.Source)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("watch source failed"))
			return
		}
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events():
				if !ok {
					return
				}
				if event.Err != nil {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(event.Err.Error()))
					return
				}
				_ = conn.WriteMessage(websocket.TextMessage, event.Data)
			}
		}
	}
}

func handleImageWebSocket(route ParsedRoute) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ApplyHeaders(w, route.Headers)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
		defer cancel()

		initial, err := os.ReadFile(route.Source)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("read source failed"))
			return
		}
		_ = conn.WriteMessage(websocket.BinaryMessage, initial)

		if !route.Watch {
			<-ctx.Done()
			return
		}

		watcher, err := newFSWatcher(route.Source)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("watch source failed"))
			return
		}
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events():
				if !ok {
					return
				}
				if event.Err != nil {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(event.Err.Error()))
					return
				}
				_ = conn.WriteMessage(websocket.BinaryMessage, event.Data)
			}
		}
	}
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var builder strings.Builder
	reader := bufio.NewReader(f)
	for {
		chunk, err := reader.ReadBytes('\n')
		builder.Write(chunk)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
	}
	return []byte(builder.String()), nil
}

func writeSSE(w io.Writer, event, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
}

func encodeBase64(data []byte) string {
	return "data:;base64," + base64.StdEncoding.EncodeToString(data)
}
