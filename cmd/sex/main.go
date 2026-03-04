package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/nep-0/sex"
)

func main() {
	configPath := flag.String("config", "sex.toml", "path to config file")
	configPathShort := flag.String("c", "sex.toml", "path to config file (shorthand)")
	flag.Parse()

	config := *configPath
	if config == "sex.toml" && *configPathShort != "sex.toml" {
		config = *configPathShort
	}

	cfg, err := sex.LoadConfig(config)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := sex.ValidateConfig(cfg); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	parsed, err := sex.ParseConfig(cfg)
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}
	server, err := sex.NewServer(parsed)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	fmt.Printf("sex listening on %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server stopped: %v", err)
	}
}
