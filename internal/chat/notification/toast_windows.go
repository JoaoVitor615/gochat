//go:build windows

package notification

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
)

func encodeUTF16LEBase64(s string) string {
	u16 := utf16.Encode([]rune(s))
	b := make([]byte, len(u16)*2)
	for i, r := range u16 {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func defaultShowToast(title, message, subtext string) {
	// 1. Emit OSC 777 and OSC 9 escape sequences for Windows Terminal and other supporting terminal emulators
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]777;notify;%s;%s\x1b\\", title, message)
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]9;%s: %s\x1b\\", title, message)

	// 2. Trigger native Windows Toast Notification in background via WinRT using -EncodedCommand
	go func() {
		xmlTitle := escapeXML(title)
		xmlMsg := escapeXML(message)
		xmlSub := escapeXML(subtext)

		xmlPayload := fmt.Sprintf("<toast><visual><binding template=\"ToastGeneric\"><text>%s</text><text>%s</text><text>%s</text></binding></visual></toast>", xmlTitle, xmlMsg, xmlSub)
		psXml := strings.ReplaceAll(xmlPayload, "'", "''")

		psScript := fmt.Sprintf(`
$regPath = 'HKCU:\Software\Classes\AppUserModelId\GoChat'
if (-not (Test-Path $regPath)) {
    New-Item -Path $regPath -Force | Out-Null
    Set-ItemProperty -Path $regPath -Name 'DisplayName' -Value 'GoChat'
}

[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml('%s')
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
try {
    $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('GoChat')
    $notifier.Show($toast)
} catch {
    $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe')
    $notifier.Show($toast)
}
`, psXml)

		encoded := encodeUTF16LEBase64(psScript)

		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-EncodedCommand", encoded)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		_ = cmd.Run()
	}()
}
