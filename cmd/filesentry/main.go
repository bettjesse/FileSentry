package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	// "filesentry/internal/config"
	// "filesentry/internal/watcher"
	"github.com/bettjesse/FileSentry/internal/config"
	"github.com/bettjesse/FileSentry/internal/watcher"
)

func main() {
	// Load YAML rules from the configs folder.
	rules, err := config.LoadRules("configs/rules.yaml")
	if err != nil {
		log.Fatalf("Rule loading failed: %v", err)
	}
	log.Printf("Successfully loaded %d rules", len(rules))

	// Set dry-run mode from environment variable and flag.
	var dryRun bool
	dryRun = os.Getenv("DRY_RUN") == "true" || os.Getenv("DRY_RUN") == "1"
	flag.BoolVar(&dryRun, "dry-run", dryRun, "Preview changes without moving files")
	flag.Parse()
	watcher.DryRun = dryRun

	// Get watch directory from environment variable (default if not set).
	watchDir := os.Getenv("WATCH_DIR")
	if watchDir == "" {
		watchDir = "./data/watcher"
	}

	// Graceful shutdown handler.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down watcher...")
		os.Exit(0)
	}()

	// Start the file watcher.
	if err := watcher.StartWatcher(rules, watchDir); err != nil {
		log.Fatalf("Watcher error: %v", err)
	}
}
