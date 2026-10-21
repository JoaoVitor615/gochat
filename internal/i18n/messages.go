// Package i18n provides type-safe internationalization and localization for GoChat.
package i18n

import (
	"time"
)

// Messages defines all user-facing strings across the application.
type Messages interface {
	// Language metadata
	LanguageName() string
	LanguageTag() string

	// Splash Screen
	SplashConnecting() string
	SplashConnected() string
	SplashError() string
	SplashQuitHint() string
	SplashContinueHint() string
	SplashLocalPeerID(id string) string

	// Terminal Window Constraints
	OpeningApp() string
	TerminalTooSmall(cols, rows int) string

	// Sidebar & Contacts
	PeersTitle() string
	AddContactButton() string
	OnlineStatus() string
	PresenceUnchecked() string
	LastSeen(t time.Time) string

	// Navigation Footers (Keyboard hints)
	FooterHome() string
	FooterChat() string
	FooterLogs() string
	FooterNickname() string
	FooterInviteNew() string
	FooterInviteExpire() string
	FooterCancel() string
	FooterSettings() string

	// Chat Notices & Operational Status
	NoticeDeliveredAck() string
	NoticeHistoryError(err error) string
	NoticeQueueError(err error) string
	NoticeSavedWaitingDelivery(err error) string
	NoticeSavedQueue() string
	NoticePeerSavedDialFailed(err error) string
	NoticeP2PConnected() string
	NoticePeerNotSaved(err error) string
	NoticeInviteAcceptedNotSaved(err error) string
	NoticeInviteAcceptedDialFailed(err error) string
	NoticeInviteAcceptedConnected() string
	NoticeEnqueuing() string
	NoticeSenderUnavailable() string
	NoticeLoadingOlder() string
	NoticeEndOfHistory() string
	NoticeNicknameUpdated() string
	NoticeNicknameRemoved() string
	NoticeWipeSuccess() string
	NoticeWipeError(err error) string

	// Message Bubbles & Chat
	StatusQueued() string
	StatusSent() string
	StatusDelivered() string
	StatusReceived() string
	LabelYou() string
	EmptyChatPrompt() string
	SendingStatus() string
	NoChatOpenTitle() string
	NoChatOpenHint() string

	// Add Contact Panel
	AddPanelTitle() string
	AddPanelPrompt() string
	AddPanelErrorLength() string
	AddPanelErrorNoPeerID() string
	InstructionAddPeer() string
	InstructionQueryingInvite() string

	// Invite Panel
	InvitePanelTitle() string
	InvitePanelPrompt() string
	InvitePanelNotice() string
	InvitePanelGenerating() string
	InvitePanelWaiting() string
	InvitePanelError(err error) string
	InstructionInvite() string
	InstructionWaitingAPI() string

	// Nickname Panel
	NicknamePanelTitle() string
	NicknamePanelPrompt(name string) string
	NicknamePanelError(err error) string
	ContactLabel() string
	NoNicknamePlaceholder() string
	InstructionNickname() string

	// Logs Panel
	LogsPanelTitle() string
	LogsPanelSubtitle() string
	LogsPanelEmpty() string

	// Settings Panel
	SettingsPanelTitle() string
	SettingsLanguageSection() string
	SettingsDangerSection() string
	SettingsWipeOption() string
	SettingsWipeConfirmTitle() string
	SettingsWipeWarning() string
	SettingsWipePrompt() string
	InstructionSettings() string
}
