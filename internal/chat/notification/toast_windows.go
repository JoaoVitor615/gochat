//go:build windows

package notification

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func defaultShowToast(title, message, subtext string) {
	// 1. Emit OSC 777 and OSC 9 escape sequences for Windows Terminal and other supporting terminal emulators
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]777;notify;%s;%s\x1b\\", title, message)
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]9;%s: %s\x1b\\", title, message)

	// 2. Trigger native Windows Toast Notification in background via WinRT
	go func() {
		xmlTitle := escapeXML(title)
		xmlMsg := escapeXML(message)
		xmlSub := escapeXML(subtext)

		psScript := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$template = @"
<toast>
    <visual>
        <binding template="ToastGeneric">
            <text>%s</text>
            <text>%s</text>
            <text>%s</text>
        </binding>
    </visual>
</toast>
"@
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("GoChat").Show($toast)
`, xmlTitle, xmlMsg, xmlSub)

		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		_ = cmd.Run()
	}()
}
