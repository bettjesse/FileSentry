package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules []Rule `yaml:"rules"`
}
type Rule struct {
	Name    string   `yaml:"name"`
	Watch   string   `yaml:"watch"`
	Filters []Filter `yaml:"filters"`
	Actions []Action `yaml:"actions"`
}
type Filter struct {
	Extension string `yaml:"extension"`
}
type Action struct {
	Move string `yaml:"move"`
}

func loadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules: %w", err)
	}
	// Parse into Config struct
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid YAML format: %w", err)
	}
	return config.Rules, nil
}
func main() {

	// Load rules from YAML file
	rules, err := loadRules("rules.yaml")
	if err != nil {
		log.Fatal("Rule loading failed:", err)
	}
	log.Printf("Successfully loaded %d rules", len(rules))
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
