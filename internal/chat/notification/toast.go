package notification

import (
	"bytes"
	"encoding/xml"
)

// ShowToastFunc defines the signature for displaying a system toast notification.
type ShowToastFunc func(title, message, subtext string)

// ShowToast is the active handler used by Send. Can be overridden in tests.
var ShowToast ShowToastFunc = defaultShowToast

// Send displays an OS native toast notification with contact title, message count, and subtext.
func Send(title, message, subtext string) {
	if ShowToast != nil {
		ShowToast(title, message, subtext)
	}
}

// escapeXML escapes text for inclusion in XML toast templates.
func escapeXML(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}
