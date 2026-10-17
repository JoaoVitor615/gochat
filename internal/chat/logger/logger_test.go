package logger

import (
	"log"
	"testing"
)

func TestLoggerBuffer(t *testing.T) {
	Reset()
	log.SetFlags(0)
	log.SetOutput(defaultBuffer)

	log.Printf("first message")
	log.Printf("second message")

	lines := Lines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "first message" {
		t.Errorf("expected 'first message', got %q", lines[0])
	}
	if lines[1] != "second message" {
		t.Errorf("expected 'second message', got %q", lines[1])
	}
}
