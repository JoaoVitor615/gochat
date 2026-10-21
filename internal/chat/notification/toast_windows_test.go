//go:build windows

package notification

import (
	"testing"
	"time"
)

func TestDefaultShowToast(t *testing.T) {
	// Call defaultShowToast directly to verify no panics and clean execution
	defaultShowToast("João", "2 mensagens novas", "Abra o terminal para visualizar")
	time.Sleep(500 * time.Millisecond)
}
