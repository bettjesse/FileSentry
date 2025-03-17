package actions

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

// SameFilesystem checks if source and destination are on the same filesystem
// by comparing the device IDs of the source file and destination directory.
func SameFilesystem(source, destDir string) (bool, error) {
	srcInfo, err := os.Stat(source)
	if err != nil {
		return false, fmt.Errorf("failed to stat source: %w", err)
	}

	destInfo, err := os.Stat(destDir)
	if err != nil {
		return false, fmt.Errorf("failed to stat destination: %w", err)
	}

	srcStat, ok := srcInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to get raw stat for source")
	}

	destStat, ok := destInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to get raw stat for destination")
	}

	return srcStat.Dev == destStat.Dev, nil
}

// RenameFile renames a file based on a regex pattern.
func RenameFile(source, pattern, replacement string) (string, error) {
	base := filepath.Base(source)
	dir := filepath.Dir(source)

	re := regexp.MustCompile(pattern)
	newName := re.ReplaceAllString(base, replacement)

	newPath := filepath.Join(dir, newName)
	err := os.Rename(source, newPath)
	return newPath, err
}

// CopyFile performs a file copy preserving permissions and timestamps.
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

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

	// Preserve the modification time.
	if err := os.Chtimes(dst, time.Now(), info.ModTime()); err != nil {
		return err
	}
	return nil
}

// RenameFileOrDir handles renaming for both files and directories.
func RenameFileOrDir(source, pattern, replacement string) (string, error) {
	base := filepath.Base(source)
	dir := filepath.Dir(source)

	re := regexp.MustCompile(pattern)
	newName := re.ReplaceAllString(base, replacement)
	newPath := filepath.Join(dir, newName)

	// Check if source exists.
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return "", fmt.Errorf("source path does not exist: %s", source)
	}

	err := os.Rename(source, newPath)
	if err != nil {
		return "", fmt.Errorf("rename failed: %w", err)
	}

	return newPath, nil
}

// MoveFile moves a file from source to destination directory.
// It uses a fast os.Rename if the files are on the same filesystem,
// otherwise it falls back to copying the file and then deleting the source.
func MoveFile(source, destDir string) error {
	log.Printf("MOVING: %s → %s", source, destDir)

	// Check if source file exists.
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", source)
	}

	// Ensure destination directory exists.
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(source))
	sameFS, err := SameFilesystem(source, destDir)
	if err != nil {
		return fmt.Errorf("filesystem check failed: %w", err)
	}

	if sameFS {
		// Fast rename when on the same filesystem.
		return os.Rename(source, destPath)
	}

	// Fall back to copy + delete when on different filesystems.
	if err := CopyFile(source, destPath); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("copy verification failed: %w", err)
	}

	if err := os.Remove(source); err != nil {
		// Clean up the destination file if deletion fails.
		os.Remove(destPath)
		return fmt.Errorf("failed to remove original: %w", err)
	}

	return nil
}
