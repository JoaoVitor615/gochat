package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/JoaoVitor615/gochat/internal/chat"
)

const maxInMemoryLogs = 500

// Buffer captures standard library log messages in memory and optionally writes to a file.
type Buffer struct {
	mu    sync.RWMutex
	lines []string
	file  *os.File
}

var defaultBuffer = &Buffer{
	lines: make([]string, 0, maxInMemoryLogs),
}

// Init sets up standard log redirection to an in-memory buffer and ~/.gochat/gochat.log.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to memory-only buffer
		log.SetFlags(0)
		log.SetOutput(defaultBuffer)
		return fmt.Errorf("resolve user home directory: %w", err)
	}

	dir := filepath.Join(home, chat.DataDirectory)
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.SetFlags(0)
		log.SetOutput(defaultBuffer)
		return fmt.Errorf("create log directory: %w", err)
	}

	logFilePath := filepath.Join(dir, "gochat.log")
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.SetFlags(0)
		log.SetOutput(defaultBuffer)
		return fmt.Errorf("open log file: %w", err)
	}

	defaultBuffer.mu.Lock()
	if defaultBuffer.file != nil {
		_ = defaultBuffer.file.Close()
	}
	defaultBuffer.file = file
	defaultBuffer.mu.Unlock()

	log.SetFlags(0)
	log.SetOutput(defaultBuffer)
	return nil
}

// Close closes the underlying log file, if open.
func Close() error {
	defaultBuffer.mu.Lock()
	defer defaultBuffer.mu.Unlock()
	if defaultBuffer.file != nil {
		err := defaultBuffer.file.Close()
		defaultBuffer.file = nil
		return err
	}
	return nil
}

// Write captures log output, appending to the log file and the in-memory buffer.
func (b *Buffer) Write(p []byte) (n int, err error) {
	raw := string(p)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.file != nil {
		_, _ = b.file.WriteString(raw)
	}

	trimmed := strings.TrimRight(raw, "\r\n")
	if trimmed != "" {
		for _, line := range strings.Split(trimmed, "\n") {
			cleanLine := strings.TrimRight(line, "\r")
			if cleanLine == "" {
				continue
			}
			if len(b.lines) >= maxInMemoryLogs {
				b.lines = b.lines[1:]
			}
			b.lines = append(b.lines, cleanLine)
		}
	}

	return len(p), nil
}

// Lines returns a snapshot copy of the recorded in-memory log lines.
func Lines() []string {
	defaultBuffer.mu.RLock()
	defer defaultBuffer.mu.RUnlock()
	result := make([]string, len(defaultBuffer.lines))
	copy(result, defaultBuffer.lines)
	return result
}

// Reset clears in-memory logs (primarily for testing).
func Reset() {
	defaultBuffer.mu.Lock()
	defer defaultBuffer.mu.Unlock()
	defaultBuffer.lines = defaultBuffer.lines[:0]
}
