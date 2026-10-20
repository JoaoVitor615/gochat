package i18n

import (
	"fmt"
	"time"
)

type esES struct{}

// EsES is the Spanish language bundle.
var EsES Messages = &esES{}

func (e *esES) LanguageName() string { return "Español" }
func (e *esES) LanguageTag() string  { return "es-ES" }

// Splash Screen
func (e *esES) SplashConnecting() string          { return "Conectando a la red P2P..." }
func (e *esES) SplashConnected() string           { return "¡Listo!" }
func (e *esES) SplashError() string               { return "Error al conectar" }
func (e *esES) SplashQuitHint() string            { return "presiona 'q' para salir" }
func (e *esES) SplashContinueHint() string        { return "presiona enter o espacio para continuar" }
func (e *esES) SplashLocalPeerID(id string) string { return "ID del peer local: " + id }

// Terminal Window Constraints
func (e *esES) OpeningApp() string { return "Abriendo gochat..." }
func (e *esES) TerminalTooSmall(cols, rows int) string {
	return fmt.Sprintf("Amplía el terminal a al menos %d columnas y %d filas.", cols, rows)
}

// Sidebar & Contacts
func (e *esES) PeersTitle() string        { return "PEERS" }
func (e *esES) AddContactButton() string  { return "+ Añadir [a]" }
func (e *esES) OnlineStatus() string      { return "en línea" }
func (e *esES) PresenceUnchecked() string { return "presencia no verificada" }
func (e *esES) LastSeen(t time.Time) string {
	if t.IsZero() {
		return e.PresenceUnchecked()
	}
	return fmt.Sprintf("visto %02d/%02d %02d:%02d", t.Day(), t.Month(), t.Hour(), t.Minute())
}

// Navigation Footers (Keyboard hints)
func (e *esES) FooterHome() string {
	return "enter abrir  ·  + invitar [i]  ·  configuración [esc]  ·  logs [ctrl+l]  ·  salir [q]"
}
func (e *esES) FooterChat() string {
	return "enter enviar  ·  desplazar historial  ·  ctrl+n apodo  ·  esc volver  ·  ctrl+u borrar"
}
func (e *esES) FooterLogs() string {
	return "esc cerrar logs  ·  re pág/av pág desplazar  ·  ctrl+l cerrar"
}
func (e *esES) FooterNickname() string {
	return "enter guardar  ·  esc cancelar"
}
func (e *esES) FooterInviteNew() string {
	return "esc cerrar  ·  r generar nuevo código"
}
func (e *esES) FooterInviteExpire() string {
	return "esc cerrar  ·  las invitaciones caducan en 10 minutos"
}
func (e *esES) FooterCancel() string {
	return "esc cancelar"
}
func (e *esES) FooterSettings() string {
	return "esc cerrar configuración  ·  enter aplicar idioma"
}

// Chat Notices & Operational Status
func (e *esES) NoticeDeliveredAck() string {
	return "Entrega confirmada por el peer."
}
func (e *esES) NoticeHistoryError(err error) string {
	if err == nil {
		return "Error al cargar el historial."
	}
	return "Error al cargar el historial: " + err.Error()
}
func (e *esES) NoticeQueueError(err error) string {
	if err == nil {
		return "No se pudo poner en cola."
	}
	return "No se pudo poner en cola: " + err.Error()
}
func (e *esES) NoticeSavedWaitingDelivery(err error) string {
	if err == nil {
		return "Guardado localmente; esperando entrega."
	}
	return "Guardado localmente; esperando entrega. " + err.Error()
}
func (e *esES) NoticeSavedQueue() string {
	return "Mensaje guardado en la cola local."
}
func (e *esES) NoticePeerSavedDialFailed(err error) string {
	if err == nil {
		return "Peer guardado; la conexión P2P directa falló."
	}
	return "Peer guardado; la conexión P2P directa falló: " + err.Error()
}
func (e *esES) NoticeP2PConnected() string {
	return "Conexión P2P directa establecida."
}
func (e *esES) NoticePeerNotSaved(err error) string {
	if err == nil {
		return "Peer añadido en esta sesión, pero no guardado localmente."
	}
	return "Peer añadido en esta sesión, pero no guardado localmente: " + err.Error()
}
func (e *esES) NoticeInviteAcceptedNotSaved(err error) string {
	if err == nil {
		return "Invitación aceptada, pero el contacto no se guardó localmente."
	}
	return "Invitación aceptada, pero el contacto no se guardó localmente: " + err.Error()
}
func (e *esES) NoticeInviteAcceptedDialFailed(err error) string {
	if err == nil {
		return "Invitación aceptada; la conexión P2P directa falló."
	}
	return "Invitación aceptada; la conexión P2P directa falló: " + err.Error()
}
func (e *esES) NoticeInviteAcceptedConnected() string {
	return "Invitación aceptada; conexión P2P directa establecida."
}
func (e *esES) NoticeEnqueuing() string {
	return "Poniendo mensaje en cola…"
}
func (e *esES) NoticeSenderUnavailable() string {
	return "El servicio de envío no está disponible."
}
func (e *esES) NoticeLoadingOlder() string {
	return "Cargando mensajes anteriores..."
}
func (e *esES) NoticeEndOfHistory() string {
	return "Inicio del historial de mensajes"
}
func (e *esES) NoticeNicknameUpdated() string {
	return "Apodo actualizado."
}
func (e *esES) NoticeNicknameRemoved() string {
	return "Apodo eliminado."
}

// Message Bubbles & Chat
func (e *esES) StatusQueued() string    { return "en cola" }
func (e *esES) StatusSent() string      { return "enviado · esperando confirmación" }
func (e *esES) StatusDelivered() string { return "entregado" }
func (e *esES) StatusReceived() string  { return "recibido" }
func (e *esES) LabelYou() string        { return "Tú" }
func (e *esES) EmptyChatPrompt() string {
	return "No hay mensajes todavía. ¡Envía el primer mensaje!"
}
func (e *esES) SendingStatus() string {
	return "enviando…"
}
func (e *esES) NoChatOpenTitle() string {
	return "Ningún peer abierto"
}
func (e *esES) NoChatOpenHint() string {
	return "Usa [i] para crear invitación o [a] para añadir contacto"
}

// Add Contact Panel
func (e *esES) AddPanelTitle() string { return "AÑADIR PEER" }
func (e *esES) AddPanelPrompt() string {
	return "Código de invitación"
}
func (e *esES) AddPanelErrorLength() string {
	return "El código debe contener 9 caracteres."
}
func (e *esES) AddPanelErrorNoPeerID() string {
	return "La API no devolvió el ID del peer."
}
func (e *esES) InstructionAddPeer() string {
	return "escribe el código · enter añadir"
}
func (e *esES) InstructionQueryingInvite() string {
	return "consultando invitación…"
}

// Invite Panel
func (e *esES) InvitePanelTitle() string { return "TU INVITACIÓN" }
func (e *esES) InvitePanelPrompt() string {
	return "Comparte este código:"
}
func (e *esES) InvitePanelNotice() string {
	return "válido por 10 minutos"
}
func (e *esES) InvitePanelGenerating() string {
	return "generando invitación…"
}
func (e *esES) InvitePanelWaiting() string {
	return "esperando que el contacto acepte…"
}
func (e *esES) InvitePanelError(err error) string {
	if err == nil {
		return "Error al consultar invitaciones aceptadas."
	}
	return "Error al consultar invitaciones aceptadas: " + err.Error()
}
func (e *esES) InstructionInvite() string {
	return "comparte el código · r nuevo · esc cerrar"
}
func (e *esES) InstructionWaitingAPI() string {
	return "esperando respuesta de la API…"
}

// Nickname Panel
func (e *esES) NicknamePanelTitle() string { return "APODO DEL CONTACTO" }
func (e *esES) NicknamePanelPrompt(name string) string {
	return fmt.Sprintf("Nuevo apodo para %s (dejar en blanco para eliminar):", name)
}
func (e *esES) NicknamePanelError(err error) string {
	if err == nil {
		return "Error al actualizar apodo."
	}
	return "Error al actualizar apodo: " + err.Error()
}
func (e *esES) ContactLabel() string {
	return "Contacto: "
}
func (e *esES) NoNicknamePlaceholder() string {
	return "sin apodo (vacío)"
}
func (e *esES) InstructionNickname() string {
	return "escribe el apodo · enter guardar · esc cancelar"
}

// Logs Panel
func (e *esES) LogsPanelTitle() string { return "REGISTROS DEL SISTEMA" }
func (e *esES) LogsPanelSubtitle() string {
	return "Registro de eventos e conexiones (ESC para cerrar)"
}
func (e *esES) LogsPanelEmpty() string { return "No hay registros hasta el momento." }

// Settings Panel
func (e *esES) SettingsPanelTitle() string       { return "CONFIGURACIÓN" }
func (e *esES) SettingsLanguageSection() string  { return "Idioma / Language:" }
func (e *esES) InstructionSettings() string      { return "↑/↓ navegar  ·  enter aplicar  ·  esc cerrar" }
