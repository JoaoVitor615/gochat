//go:build !windows

package notification

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func defaultShowToast(title, message, subtext string) {
	// 1. Emit OSC 777 and OSC 9 escape sequences for desktop terminal emulators
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]777;notify;%s;%s\x1b\\", title, message)
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]9;%s: %s\x1b\\", title, message)

	go func() {
		switch runtime.GOOS {
		case "darwin":
			body := message
			if subtext != "" {
				body += " (" + subtext + ")"
			}
			script := fmt.Sprintf(`display notification %q with title %q`, body, title)
			_ = exec.Command("osascript", "-e", script).Run()
		case "linux":
			body := message
			if subtext != "" {
				body += "\n" + subtext
			}
			_ = exec.Command("notify-send", title, body).Run()
		}
	}()
}
