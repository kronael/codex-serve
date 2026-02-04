package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "path to config file")
	jwtCmd := flag.Bool("jwt", false, "JWT subcommand")
	flag.Parse()

	// Handle JWT subcommand
	if *jwtCmd {
		handleJWT(flag.Args())
		return
	}

	// Load config
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create server
	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	// Start server
	log.Printf("starting server on %s", cfg.Address)
	if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

// handleJWT handles JWT subcommands: secret, token
func handleJWT(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: codex-serve -jwt <secret|token> [args]")
		os.Exit(1)
	}

	switch args[0] {
	case "secret":
		secret, err := GenerateSecret()
		if err != nil {
			log.Fatalf("failed to generate secret: %v", err)
		}
		fmt.Println(secret)

	case "token":
		if len(args) < 2 {
			fmt.Println("usage: codex-serve -jwt token <secret>")
			os.Exit(1)
		}
		secret := args[1]
		token, err := GenerateToken(secret, "codex-serve", 24*time.Hour)
		if err != nil {
			log.Fatalf("failed to generate token: %v", err)
		}
		fmt.Println(token)

	default:
		fmt.Printf("unknown jwt command: %s\n", args[0])
		os.Exit(1)
	}
}
