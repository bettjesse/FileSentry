package actions

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// SameFilesystem checks if source and destination are on the same filesystem.
func SameFilesystem(source, dest string) (bool, error) {
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

// CopyFile performs a robust file copy preserving permissions and timestamps.
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

	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return nil
}

// MoveFile moves a file from source to destination directory.
// It uses rename when possible or falls back to copy+delete.
func MoveFile(source, destDir string) error {
	log.Printf("MOVING: %s → %s", source, destDir)
	maxRetries := 5
	retryDelay := 200 * time.Millisecond

	// Retry if the file vanishes momentarily.
	for i := 0; i < maxRetries; i++ {
		if _, err := os.Stat(source); os.IsNotExist(err) {
			time.Sleep(retryDelay + time.Duration(rand.Intn(50))*time.Millisecond)
			continue
		}
		break
	}
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("file vanished after retries: %s", source)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(source))
	sameFS, err := SameFilesystem(source, destDir)
	if err != nil {
		return fmt.Errorf("filesystem check failed: %w", err)
	}

	if sameFS {
		return os.Rename(source, destPath)
	}

	if err := CopyFile(source, destPath); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("copy verification failed: %w", err)
	}

	if err := os.Remove(source); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("failed to remove original: %w", err)
	}

	return nil
}
