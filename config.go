package sex

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server ServerConfig  `toml:"server"`
	Routes []RouteConfig `toml:"route"`
}

type ServerConfig struct {
	Address      string `toml:"address"`
	ReadTimeout  string `toml:"read_timeout"`
	WriteTimeout string `toml:"write_timeout"`
}

type RouteConfig struct {
	Path         string            `toml:"path"`
	ExposeType   string            `toml:"expose_type"`
	ResourceType string            `toml:"resource_type"`
	Source       string            `toml:"source"`
	Headers      map[string]string `toml:"headers"`
	Watch        bool              `toml:"watch"`
	Timeout      string            `toml:"timeout"`
}

type ParsedRoute struct {
	RouteConfig
	Timeout time.Duration
}

type ParsedConfig struct {
	Config
	Routes []ParsedRoute
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := UnmarshalTOML(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse toml: %w", err)
	}
	return cfg, nil
}

func ParseConfig(cfg Config) (ParsedConfig, error) {
	parsed := ParsedConfig{Config: cfg}
	for i, route := range cfg.Routes {
		timeout := 30 * time.Second
		if route.Timeout != "" {
			d, err := time.ParseDuration(route.Timeout)
			if err != nil {
				return parsed, fmt.Errorf("route[%d]: parse timeout: %w", i, err)
			}
			timeout = d
		}
		parsed.Routes = append(parsed.Routes, ParsedRoute{
			RouteConfig: route,
			Timeout:     timeout,
		})
	}
	return parsed, nil
}
