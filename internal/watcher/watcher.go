package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/bettjesse/FileSentry/internal/actions"
	"github.com/bettjesse/FileSentry/internal/config"
)

// DryRun is a package-level flag to simulate actions.
var DryRun bool

// MatchesFilters checks whether the file's extension matches any of the filters.
func MatchesFilters(filePath string, filters []config.Filter) bool {
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

// IsTrashPath checks common trash paths.
func IsTrashPath(path string) bool {
	return strings.Contains(path, "/Trash/") ||
		strings.Contains(path, "/.Trash/") ||
		strings.Contains(path, "/.local/share/Trash/") ||
		strings.Contains(path, "/$RECYCLE.BIN/")
}

// HandleEvent processes a single fsnotify event.
func HandleEvent(event fsnotify.Event, rules []config.Rule) {
	// Debounce create/write events.
	if event.Op&fsnotify.Create == fsnotify.Create {
		time.Sleep(300 * time.Millisecond)
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		time.Sleep(100 * time.Millisecond)
	}

	for _, rule := range rules {
		if MatchesFilters(event.Name, rule.Filters) {
			for _, action := range rule.Actions {
				if action.Move != "" {
					if DryRun {
						log.Printf("DRY-RUN: Moving %s to %s", event.Name, action.Move)
					} else {
						if err := actions.MoveFile(event.Name, action.Move); err != nil {
							log.Printf("ERROR: %v", err)
						}
					}
				}
			}
		}
	}

	// Log the type of event.
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		log.Printf("Created: %s", event.Name)
	case event.Op&fsnotify.Write == fsnotify.Write:
		log.Printf("Modified: %s", event.Name)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		if IsTrashPath(event.Name) {
			log.Printf("Moved to trash: %s", event.Name)
		} else {
			log.Printf("Renamed: %s", event.Name)
		}
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		log.Printf("Permanently deleted: %s", event.Name)
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		log.Printf("Permissions changed: %s", event.Name)
	default:
		log.Printf("Unknown operation: %s", event.Name)
	}
}

// StartWatcher creates an fsnotify watcher, adds the specified directory, and starts the event loop.
func StartWatcher(rules []config.Rule, watchDir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if _, err := os.Stat(watchDir); os.IsNotExist(err) {
		return err
	}

	if err := watcher.Add(watchDir); err != nil {
		return err
	}
	log.Printf("Watching directory: %s", watchDir)

	// Main event loop.
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Println("Watcher events channel closed")
				return nil
			}
			HandleEvent(event, rules)
		case err, ok := <-watcher.Errors:
			if !ok {
				log.Println("Watcher errors channel closed")
				return nil
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
