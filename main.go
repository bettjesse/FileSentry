package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

var (
	permanentDelete bool // Set via CLI flag
)

func main() {
	// 1. Set up watcher with environment variable
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create watcher:", err)
	}
	defer watcher.Close()
	log.Println("Watcher created successfully")

	// 2. Get watch directory from environment variable
	watchDir := os.Getenv("WATCH_DIR")
	if watchDir == "" {
		watchDir = "./watcher" // default directory
	}

	// 3. Add directory to watch
	err = watcher.Add(watchDir)
	if err != nil {
		log.Fatal("Failed to add watch directory:", err)
	}
	log.Printf("Watching directory: %s", watchDir)

	// 4. Event loop with proper error handling
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok { // channel was closed
					log.Println("Watcher events channel closed")
					return
				}
				handleEvent(event)

			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("Watcher errors channel closed")
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	// New: Add CLI flag

	flag.Parse()

	// 5. Graceful shutdown handler
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan
	log.Println("\nShutting down watcher...")
}

func handleEvent(event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create):
		log.Printf("Created: %s", event.Name)
	case event.Has(fsnotify.Write):
		log.Printf("Modified: %s", event.Name)
	case event.Has(fsnotify.Rename):
		if isTrashPath(event.Name) {
			log.Printf("Moved to trash: %s", event.Name)
		} else {
			log.Printf("Renamed: %s", event.Name)
		}
	case event.Has(fsnotify.Remove):
		log.Printf("Permanently deleted: %s", event.Name)
	case event.Has(fsnotify.Chmod):
		log.Printf("Permissions changed: %s", event.Name)
	default:
		log.Printf("Unknown operation: %s", event.Name)
	}
}

func isTrashPath(path string) bool {
	// Common trash locations across OSes
	return strings.Contains(path, "/Trash/") ||
		strings.Contains(path, "/.Trash/") ||
		strings.Contains(path, "/.local/share/Trash/") ||
		strings.Contains(path, "/$RECYCLE.BIN/")
}
