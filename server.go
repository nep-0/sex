package sex

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func NewServer(cfg ParsedConfig) (*http.Server, error) {
	mux := http.NewServeMux()
	if err := RegisterUI(mux, cfg); err != nil {
		return nil, fmt.Errorf("ui setup failed: %w", err)
	}
	for i, route := range cfg.Routes {
		h, err := buildRouteHandler(route)
		if err != nil {
			return nil, fmt.Errorf("route[%d]: %w", i, err)
		}
		path := route.Path
		mux.HandleFunc(path, h)
		if route.ExposeType == string(ExposeSSE) && !strings.HasSuffix(path, "/") {
			mux.HandleFunc(path+"/", h)
		}
	}

	readTimeout, writeTimeout, err := parseServerTimeouts(cfg.Server)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	return srv, nil
}

func parseServerTimeouts(cfg ServerConfig) (time.Duration, time.Duration, error) {
	var readTimeout time.Duration
	var writeTimeout time.Duration
	var err error
	if cfg.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(cfg.ReadTimeout)
		if err != nil {
			return 0, 0, fmt.Errorf("parse read_timeout: %w", err)
		}
	}
	if cfg.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(cfg.WriteTimeout)
		if err != nil {
			return 0, 0, fmt.Errorf("parse write_timeout: %w", err)
		}
	}
	return readTimeout, writeTimeout, nil
}
