package sex

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrEmptyPath       = errors.New("route path is empty")
	ErrInvalidPath     = errors.New("route path must start with /")
	ErrEmptySource     = errors.New("route source is empty")
	ErrUnsupportedType = errors.New("unsupported expose or resource type")
)

func ValidateConfig(cfg Config) error {
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}
	for i, route := range cfg.Routes {
		if route.Path == "" {
			return fmt.Errorf("route[%d]: %w", i, ErrEmptyPath)
		}
		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route[%d]: %w", i, ErrInvalidPath)
		}
		if route.Source == "" {
			return fmt.Errorf("route[%d]: %w", i, ErrEmptySource)
		}
		if !isSupportedExpose(route.ExposeType) {
			return fmt.Errorf("route[%d]: %w: %s", i, ErrUnsupportedType, route.ExposeType)
		}
		if !isSupportedResource(route.ResourceType) {
			return fmt.Errorf("route[%d]: %w: %s", i, ErrUnsupportedType, route.ResourceType)
		}
		if strings.Contains(route.Path, " ") {
			return fmt.Errorf("route[%d]: path contains spaces", i)
		}
		if route.ExposeType == string(ExposeWebSocket) && route.Timeout == "" {
			// WebSocket defaults already applied in ParseConfig.
		}
		if route.ResourceType == string(ResourceImage) && route.ExposeType == string(ExposeHTTP) {
			// Still supported: serve image bytes over HTTP
			continue
		}
	}
	return nil
}

func isSupportedExpose(value string) bool {
	switch strings.ToLower(value) {
	case string(ExposeHTTP), string(ExposeSSE), string(ExposeWebSocket):
		return true
	default:
		return false
	}
}

func isSupportedResource(value string) bool {
	switch strings.ToLower(value) {
	case string(ResourceFile), string(ResourceImage):
		return true
	default:
		return false
	}
}

func ApplyHeaders(w http.ResponseWriter, headers map[string]string) {
	for key, value := range headers {
		if key == "" {
			continue
		}
		w.Header().Set(key, value)
	}
}
