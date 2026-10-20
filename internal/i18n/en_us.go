package i18n

import (
	"fmt"
	"time"
)

type enUS struct{}

// EnUS is the English (United States) bundle.
var EnUS Messages = &enUS{}

func (e *enUS) LanguageName() string { return "English" }
func (e *enUS) LanguageTag() string  { return "en-US" }

// Splash Screen
func (e *enUS) SplashConnecting() string          { return "Connecting to P2P network..." }
func (e *enUS) SplashConnected() string           { return "Ready!" }
func (e *enUS) SplashError() string               { return "Connection error" }
func (e *enUS) SplashQuitHint() string            { return "press 'q' to quit" }
func (e *enUS) SplashContinueHint() string        { return "press enter or space to continue" }
func (e *enUS) SplashLocalPeerID(id string) string { return "Local Peer ID: " + id }

// Terminal Window Constraints
func (e *enUS) OpeningApp() string { return "Opening gochat..." }
func (e *enUS) TerminalTooSmall(cols, rows int) string {
	return fmt.Sprintf("Widen terminal to at least %d columns and %d rows.", cols, rows)
}

// Sidebar & Contacts
func (e *enUS) PeersTitle() string        { return "PEERS" }
func (e *enUS) AddContactButton() string  { return "+ Add [a]" }
func (e *enUS) OnlineStatus() string      { return "online" }
func (e *enUS) PresenceUnchecked() string { return "presence unverified" }
func (e *enUS) LastSeen(t time.Time) string {
	if t.IsZero() {
		return e.PresenceUnchecked()
	}
	return fmt.Sprintf("seen %02d/%02d %02d:%02d", t.Month(), t.Day(), t.Hour(), t.Minute())
}

// Navigation Footers (Keyboard hints)
func (e *enUS) FooterHome() string {
	return "enter open  ·  + invite [i]  ·  settings [esc]  ·  logs [ctrl+l]  ·  quit [q]"
}
func (e *enUS) FooterChat() string {
	return "enter send  ·  scroll history  ·  ctrl+n nickname  ·  esc back  ·  ctrl+u clear"
}
func (e *enUS) FooterLogs() string {
	return "esc close logs  ·  pgup/pgdown scroll  ·  ctrl+l close"
}
func (e *enUS) FooterNickname() string {
	return "enter save  ·  esc cancel"
}
func (e *enUS) FooterInviteNew() string {
	return "esc close  ·  r generate new code"
}
func (e *enUS) FooterInviteExpire() string {
	return "esc close  ·  invites expire after 10 minutes"
}
func (e *enUS) FooterCancel() string {
	return "esc cancel"
}
func (e *enUS) FooterSettings() string {
	return "esc close settings  ·  enter apply language"
}

// Chat Notices & Operational Status
func (e *enUS) NoticeDeliveredAck() string {
	return "Delivered and acknowledged by peer."
}
func (e *enUS) NoticeHistoryError(err error) string {
	if err == nil {
		return "Failed to load history."
	}
	return "Failed to load history: " + err.Error()
}
func (e *enUS) NoticeQueueError(err error) string {
	if err == nil {
		return "Failed to queue message."
	}
	return "Failed to queue message: " + err.Error()
}
func (e *enUS) NoticeSavedWaitingDelivery(err error) string {
	if err == nil {
		return "Saved locally; awaiting delivery."
	}
	return "Saved locally; awaiting delivery. " + err.Error()
}
func (e *enUS) NoticeSavedQueue() string {
	return "Message saved to local outbox queue."
}
func (e *enUS) NoticePeerSavedDialFailed(err error) string {
	if err == nil {
		return "Peer saved; direct P2P dial failed."
	}
	return "Peer saved; direct P2P dial failed: " + err.Error()
}
func (e *enUS) NoticeP2PConnected() string {
	return "Direct P2P connection established."
}
func (e *enUS) NoticePeerNotSaved(err error) string {
	if err == nil {
		return "Peer added for this session, but could not be saved locally."
	}
	return "Peer added for this session, but could not be saved locally: " + err.Error()
}
func (e *enUS) NoticeInviteAcceptedNotSaved(err error) string {
	if err == nil {
		return "Invite accepted, but contact could not be saved locally."
	}
	return "Invite accepted, but contact could not be saved locally: " + err.Error()
}
func (e *enUS) NoticeInviteAcceptedDialFailed(err error) string {
	if err == nil {
		return "Invite accepted; direct P2P dial failed."
	}
	return "Invite accepted; direct P2P dial failed: " + err.Error()
}
func (e *enUS) NoticeInviteAcceptedConnected() string {
	return "Invite accepted; direct P2P connection established."
}
func (e *enUS) NoticeEnqueuing() string {
	return "Enqueuing message…"
}
func (e *enUS) NoticeSenderUnavailable() string {
	return "Message sending service is unavailable."
}
func (e *enUS) NoticeLoadingOlder() string {
	return "Loading older messages..."
}
func (e *enUS) NoticeEndOfHistory() string {
	return "Beginning of conversation history"
}
func (e *enUS) NoticeNicknameUpdated() string {
	return "Nickname updated."
}
func (e *enUS) NoticeNicknameRemoved() string {
	return "Nickname removed."
}

// Message Bubbles & Chat
func (e *enUS) StatusQueued() string    { return "queued" }
func (e *enUS) StatusSent() string      { return "sent · awaiting ack" }
func (e *enUS) StatusDelivered() string { return "delivered" }
func (e *enUS) StatusReceived() string  { return "received" }
func (e *enUS) LabelYou() string        { return "You" }
func (e *enUS) EmptyChatPrompt() string {
	return "No messages yet. Send the first message!"
}
func (e *enUS) SendingStatus() string {
	return "sending…"
}
func (e *enUS) NoChatOpenTitle() string {
	return "No peer chat open"
}
func (e *enUS) NoChatOpenHint() string {
	return "Use [i] to create invite or [a] to add contact"
}

// Add Contact Panel
func (e *enUS) AddPanelTitle() string { return "ADD PEER" }
func (e *enUS) AddPanelPrompt() string {
	return "Invite code"
}
func (e *enUS) AddPanelErrorLength() string {
	return "Code must contain exactly 9 characters."
}
func (e *enUS) AddPanelErrorNoPeerID() string {
	return "Discovery API did not return peer ID."
}
func (e *enUS) InstructionAddPeer() string {
	return "type code · enter to add"
}
func (e *enUS) InstructionQueryingInvite() string {
	return "querying invite…"
}

// Invite Panel
func (e *enUS) InvitePanelTitle() string { return "YOUR INVITE" }
func (e *enUS) InvitePanelPrompt() string {
	return "Share this code:"
}
func (e *enUS) InvitePanelNotice() string {
	return "valid for 10 minutes"
}
func (e *enUS) InvitePanelGenerating() string {
	return "generating invite…"
}
func (e *enUS) InvitePanelWaiting() string {
	return "waiting for contact to accept…"
}
func (e *enUS) InvitePanelError(err error) string {
	if err == nil {
		return "Failed to query accepted invites."
	}
	return "Failed to query accepted invites: " + err.Error()
}
func (e *enUS) InstructionInvite() string {
	return "share code · r new · esc close"
}
func (e *enUS) InstructionWaitingAPI() string {
	return "waiting for API response…"
}

// Nickname Panel
func (e *enUS) NicknamePanelTitle() string { return "CONTACT NICKNAME" }
func (e *enUS) NicknamePanelPrompt(name string) string {
	return fmt.Sprintf("New nickname for %s (leave empty to remove):", name)
}
func (e *enUS) NicknamePanelError(err error) string {
	if err == nil {
		return "Failed to update nickname."
	}
	return "Failed to update nickname: " + err.Error()
}
func (e *enUS) ContactLabel() string {
	return "Contact: "
}
func (e *enUS) NoNicknamePlaceholder() string {
	return "no nickname (empty)"
}
func (e *enUS) InstructionNickname() string {
	return "type nickname · enter save · esc cancel"
}

// Logs Panel
func (e *enUS) LogsPanelTitle() string { return "SYSTEM LOGS" }
func (e *enUS) LogsPanelSubtitle() string {
	return "Event and connection logs (ESC to close)"
}
func (e *enUS) LogsPanelEmpty() string { return "No logs recorded yet." }

// Settings Panel
func (e *enUS) SettingsPanelTitle() string       { return "SETTINGS" }
func (e *enUS) SettingsLanguageSection() string  { return "Language / Idioma:" }
func (e *enUS) InstructionSettings() string      { return "↑/↓ select  ·  enter apply  ·  esc close" }
