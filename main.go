package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"math/rand"

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
	Extensions []string `yaml:"extension"`
}
type Action struct {
	Move string `yaml:"move"`
}

var (
	rules  []Rule // Global rules variable
	dryRun bool   // Global dry-run flag
)

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

func matchesFilters(filePath string, filters []Filter) bool {
	ext := filepath.Ext(filePath)
	for _, filter := range filters {
		for _, ruleExt := range filter.Extensions {
			if ext == ruleExt {
				return true
			}
		}
	}
	return false
}

// Add filesystem check function
func sameFilesystem(source, dest string) (bool, error) {
	var srcStat syscall.Statfs_t
	if err := syscall.Statfs(filepath.Dir(source), &srcStat); err != nil {
		return false, fmt.Errorf("failed to stat source: %w", err)
	}

	var destStat syscall.Statfs_t
	if err := syscall.Statfs(filepath.Dir(dest), &destStat); err != nil {
		return false, fmt.Errorf("failed to stat destination: %w", err)
	}

	return srcStat.Fsid == destStat.Fsid, nil
}

// Add robust copy function
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	// Preserve permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	dest, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return err
	}

	// Preserve timestamps
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return err
	}

	return nil
}

// Updated moveFile function
func moveFile(source, destDir string) error {
	log.Printf("MOVING: %s → %s", source, destDir) // Add this line
	// Retry mechanism for vanished files
	// In moveFile()
	maxRetries := 5                      // Increased from 3 (works for WSL and normal cases)
	retryDelay := 200 * time.Millisecond // Increased from 50ms

	for i := 0; i < maxRetries; i++ {
		if _, err := os.Stat(source); os.IsNotExist(err) {
			time.Sleep(retryDelay + time.Duration(rand.Intn(50))*time.Millisecond) // Jitter
			continue
		}
		break
	}
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("file vanished after retries: %s", source)
	}

	// Create destination directory first
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(source))

	// Check if same filesystem
	sameFS, err := sameFilesystem(source, destDir)
	if err != nil {
		return fmt.Errorf("filesystem check failed: %w", err)
	}

	if sameFS {
		// Atomic rename for same filesystem
		return os.Rename(source, destPath)
	}

	// Cross-filesystem: copy + delete
	if err := copyFile(source, destPath); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// Verify copy before deleting original
	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("copy verification failed: %w", err)
	}

	if err := os.Remove(source); err != nil {
		// Preserve original if delete fails
		os.Remove(destPath) // Cleanup partial copy
		return fmt.Errorf("failed to remove original: %w", err)
	}

	return nil
}

func main() {

	// Load rules from YAML file
	var err error
	rules, err = loadRules("rules.yaml")
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

	if _, err := os.Stat(watchDir); os.IsNotExist(err) {
		log.Fatalf("Watch directory does not exist: %s (resolved from %s)",
			watchDir,
			os.Getenv("WATCH_DIR"))
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
	// **Read DRY_RUN from environment**
	// Set dry-run from environment first
	dryRun = os.Getenv("DRY_RUN") == "true" || os.Getenv("DRY_RUN") == "1"

	// Then parse flags (will override environment variable if specified)
	flag.BoolVar(&dryRun, "dry-run", dryRun, "Preview changes without moving files")
	log.Printf("DRY_RUN environment variable: %s", os.Getenv("DRY_RUN"))

	flag.Parse()

	// 5. Graceful shutdown handler
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan
	log.Println("\nShutting down watcher...")
}

func handleEvent(event fsnotify.Event) {
	// Debounce only for create/write events
	switch {
	case event.Has(fsnotify.Create):
		time.Sleep(300 * time.Millisecond) // Allow OS to stabilize new files
	case event.Has(fsnotify.Write):
		time.Sleep(100 * time.Millisecond)
	}
	for _, rule := range rules {
		if matchesFilters(event.Name, rule.Filters) {
			for _, action := range rule.Actions {
				if action.Move != "" {
					if dryRun {
						log.Printf("DRY-RUN: Moving %s to %s", event.Name, action.Move)
						continue // Skip moving files in dry-run mode
					} else {
						if err := moveFile(event.Name, action.Move); err != nil {
							log.Printf("ERROR: %v", err)
						}
					}
				}
			}
		}
	}
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
