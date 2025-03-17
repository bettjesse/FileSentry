package watcher

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/bettjesse/FileSentry/internal/actions"
	"github.com/bettjesse/FileSentry/internal/config"
)

// DryRun is a package-level flag to simulate actions.
var DryRun bool

func MatchesFilters(filePath string, filters []config.Filter) bool {
	if len(filters) == 0 {
		return true
	}

	result := true
	for _, filter := range filters {
		match := true

		// Extension check
		if len(filter.Extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(filePath))
			found := false
			for _, e := range filter.Extensions {
				if ext == strings.ToLower(e) {
					found = true
					break
				}
			}
			match = match && found
		}

		// Exclude pattern
		if filter.Exclude != "" {
			re := regexp.MustCompile(filter.Exclude)
			match = match && !re.MatchString(filepath.Base(filePath))
		}

		// Update the last_modified filter logic
		if filter.LastModified != "" {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				log.Printf("ERROR: Could not get file info for %s: %v", filePath, err)
				return false
			}

			duration, err := time.ParseDuration(filter.LastModified)
			if err != nil {
				log.Printf("ERROR: Invalid duration format %s: %v", filter.LastModified, err)
				return false
			}

			fileAge := time.Since(fileInfo.ModTime())

			switch filter.Operator {
			case "OLDER_THAN":
				match = match && (fileAge > duration)
			case "WITHIN":
				match = match && (fileAge < duration)
			default: // Default to "WITHIN" if operator not specified
				match = match && (fileAge < duration)
			}
		}
		// Combine results using operator
		if filter.Operator == "OR" {
			result = result || match
		} else { // default AND
			result = result && match
		}
	}

	return result
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

	// Check if the file exists before processing
	if _, err := os.Stat(event.Name); os.IsNotExist(err) {
		log.Printf("Skipping event for non-existent file: %s", event.Name)
		return
	}

	for _, rule := range rules {
		if MatchesFilters(event.Name, rule.Filters) {
			for _, action := range rule.Actions {
				// Handle file renaming first.
				if action.Regex != "" {
					newPath, err := actions.RenameFileOrDir(event.Name, action.Regex, action.Replace)
					if err != nil {
						log.Printf("ERROR renaming: %v", err)
						continue
					}
					event.Name = newPath
					log.Printf("Renamed to: %s", newPath)

					// If it's a directory, process its contents.
					if info, err := os.Stat(newPath); err == nil && info.IsDir() {
						log.Printf("Processing directory contents: %s", newPath)
						filepath.Walk(newPath, func(path string, info os.FileInfo, err error) error {
							if !info.IsDir() {
								HandleEvent(fsnotify.Event{
									Name: path,
									Op:   fsnotify.Create,
								}, rules)
							}
							return nil
						})
					}
				}

				if action.Move != "" {
					if DryRun {
						log.Printf("DRY-RUN: Moving %s to %s", event.Name, action.Move)
					} else {
						if err := actions.MoveFile(event.Name, action.Move); err != nil {
							log.Printf("ERROR moving file: %v", err)
						}
					}
				}
			}
			// Once a matching rule is processed, break out to avoid duplicate processing.
			break
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

	// Collect all unique watch paths from rules
	watchDirs := make(map[string]struct{})
	for _, rule := range rules {
		watchDirs[rule.Watch] = struct{}{}
	}

	// Add all directories to watcher
	for dir := range watchDirs {
		if err := watcher.Add(dir); err != nil {
			log.Printf("WARNING: Couldn't watch %s: %v", dir, err)
		} else {
			log.Printf("Now watching: %s", dir)
		}
	}

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
