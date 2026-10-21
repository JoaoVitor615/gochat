package notification

import (
	"testing"
)

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Alice & Bob", "Alice &amp; Bob"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quotes"`, "&#34;quotes&#34;"},
	}

	for _, tc := range tests {
		got := escapeXML(tc.input)
		if got != tc.expected {
			t.Errorf("escapeXML(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestSend_CustomHook(t *testing.T) {
	orig := ShowToast
	defer func() { ShowToast = orig }()

	var capturedTitle, capturedMsg, capturedSub string
	called := false

	ShowToast = func(title, message, subtext string) {
		called = true
		capturedTitle = title
		capturedMsg = message
		capturedSub = subtext
	}

	Send("João", "2 mensagens novas", "Abra o terminal para visualizar")

	if !called {
		t.Fatal("expected ShowToast to be called")
	}
	if capturedTitle != "João" || capturedMsg != "2 mensagens novas" || capturedSub != "Abra o terminal para visualizar" {
		t.Errorf("unexpected arguments captured: title=%q, msg=%q, sub=%q", capturedTitle, capturedMsg, capturedSub)
	}
}
