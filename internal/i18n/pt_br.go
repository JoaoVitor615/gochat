package i18n

import (
	"fmt"
	"time"
)

type ptBR struct{}

// PtBR is the Brazilian Portuguese bundle.
var PtBR Messages = &ptBR{}

func (p *ptBR) LanguageName() string { return "Português (Brasil)" }
func (p *ptBR) LanguageTag() string  { return "pt-BR" }

// Splash Screen
func (p *ptBR) SplashConnecting() string          { return "Conectando à rede P2P..." }
func (p *ptBR) SplashConnected() string           { return "Pronto!" }
func (p *ptBR) SplashError() string               { return "Erro ao conectar" }
func (p *ptBR) SplashQuitHint() string            { return "pressione 'q' para sair" }
func (p *ptBR) SplashContinueHint() string        { return "pressione enter ou espaço para continuar" }
func (p *ptBR) SplashLocalPeerID(id string) string { return "ID do peer local: " + id }

// Terminal Window Constraints
func (p *ptBR) OpeningApp() string { return "Abrindo gochat..." }
func (p *ptBR) TerminalTooSmall(cols, rows int) string {
	return fmt.Sprintf("Amplie o terminal para pelo menos %d colunas e %d linhas.", cols, rows)
}

// Sidebar & Contacts
func (p *ptBR) PeersTitle() string        { return "PEERS" }
func (p *ptBR) AddContactButton() string  { return "+ Adicionar [a]" }
func (p *ptBR) OnlineStatus() string      { return "online" }
func (p *ptBR) PresenceUnchecked() string { return "presença não verificada" }
func (p *ptBR) LastSeen(t time.Time) string {
	if t.IsZero() {
		return p.PresenceUnchecked()
	}
	return fmt.Sprintf("visto %02d/%02d %02d:%02d", t.Day(), t.Month(), t.Hour(), t.Minute())
}

// Navigation Footers (Keyboard hints)
func (p *ptBR) FooterHome() string {
	return "enter abrir  ·  + convite [i]  ·  configurações [esc]  ·  logs [ctrl+l]  ·  sair [q]"
}
func (p *ptBR) FooterChat() string {
	return "enter enviar  ·  scroll histórico  ·  ctrl+n apelido  ·  esc voltar  ·  ctrl+u limpar"
}
func (p *ptBR) FooterLogs() string {
	return "esc fechar logs  ·  pgup/pgdown scroll  ·  ctrl+l fechar"
}
func (p *ptBR) FooterNickname() string {
	return "enter salvar  ·  esc cancelar"
}
func (p *ptBR) FooterInviteNew() string {
	return "esc fechar  ·  r gerar novo código"
}
func (p *ptBR) FooterInviteExpire() string {
	return "esc fechar  ·  convites expiram após 10 minutos"
}
func (p *ptBR) FooterCancel() string {
	return "esc cancelar"
}
func (p *ptBR) FooterSettings() string {
	return "esc fechar configurações  ·  enter aplicar idioma"
}

// Chat Notices & Operational Status
func (p *ptBR) NoticeDeliveredAck() string {
	return "Entrega confirmada pelo peer."
}
func (p *ptBR) NoticeHistoryError(err error) string {
	if err == nil {
		return "Falha ao carregar histórico."
	}
	return "Falha ao carregar histórico: " + err.Error()
}
func (p *ptBR) NoticeQueueError(err error) string {
	if err == nil {
		return "Não foi possível enfileirar."
	}
	return "Não foi possível enfileirar: " + err.Error()
}
func (p *ptBR) NoticeSavedWaitingDelivery(err error) string {
	if err == nil {
		return "Salva localmente; aguardando entrega."
	}
	return "Salva localmente; aguardando entrega. " + err.Error()
}
func (p *ptBR) NoticeSavedQueue() string {
	return "Mensagem salva na fila local."
}
func (p *ptBR) NoticePeerSavedDialFailed(err error) string {
	if err == nil {
		return "Peer salvo; conexão P2P direta falhou."
	}
	return "Peer salvo; conexão P2P direta falhou: " + err.Error()
}
func (p *ptBR) NoticeP2PConnected() string {
	return "Conexão P2P direta estabelecida."
}
func (p *ptBR) NoticePeerNotSaved(err error) string {
	if err == nil {
		return "Peer adicionado nesta sessão, mas não salvo localmente."
	}
	return "Peer adicionado nesta sessão, mas não salvo localmente: " + err.Error()
}
func (p *ptBR) NoticeInviteAcceptedNotSaved(err error) string {
	if err == nil {
		return "Convite aceito, mas o contato não foi salvo localmente."
	}
	return "Convite aceito, mas o contato não foi salvo localmente: " + err.Error()
}
func (p *ptBR) NoticeInviteAcceptedDialFailed(err error) string {
	if err == nil {
		return "Convite aceito; conexão P2P direta falhou."
	}
	return "Convite aceito; conexão P2P direta falhou: " + err.Error()
}
func (p *ptBR) NoticeInviteAcceptedConnected() string {
	return "Convite aceito; conexão P2P direta estabelecida."
}
func (p *ptBR) NoticeEnqueuing() string {
	return "Enfileirando mensagem…"
}
func (p *ptBR) NoticeSenderUnavailable() string {
	return "O serviço de envio não está disponível."
}
func (p *ptBR) NoticeLoadingOlder() string {
	return "Carregando mensagens anteriores..."
}
func (p *ptBR) NoticeEndOfHistory() string {
	return "Início do histórico de mensagens"
}
func (p *ptBR) NoticeNicknameUpdated() string {
	return "Apelido atualizado."
}
func (p *ptBR) NoticeNicknameRemoved() string {
	return "Apelido removido."
}

// Message Bubbles & Chat
func (p *ptBR) StatusQueued() string    { return "na fila" }
func (p *ptBR) StatusSent() string      { return "enviado · aguardando confirmação" }
func (p *ptBR) StatusDelivered() string { return "entregue" }
func (p *ptBR) StatusReceived() string  { return "recebida" }
func (p *ptBR) LabelYou() string        { return "Você" }
func (p *ptBR) EmptyChatPrompt() string {
	return "Nenhuma mensagem ainda. Envie a primeira mensagem!"
}
func (p *ptBR) SendingStatus() string {
	return "enviando…"
}
func (p *ptBR) NoChatOpenTitle() string {
	return "Nenhum peer aberto"
}
func (p *ptBR) NoChatOpenHint() string {
	return "Use [i] para criar convite ou [a] para adicionar"
}

// Add Contact Panel
func (p *ptBR) AddPanelTitle() string { return "ADICIONAR PEER" }
func (p *ptBR) AddPanelPrompt() string {
	return "Código de convite"
}
func (p *ptBR) AddPanelErrorLength() string {
	return "O código deve conter 9 caracteres."
}
func (p *ptBR) AddPanelErrorNoPeerID() string {
	return "A API não retornou o ID do peer."
}
func (p *ptBR) InstructionAddPeer() string {
	return "digite o código · enter adicionar"
}
func (p *ptBR) InstructionQueryingInvite() string {
	return "consultando convite…"
}

// Invite Panel
func (p *ptBR) InvitePanelTitle() string { return "SEU CONVITE" }
func (p *ptBR) InvitePanelPrompt() string {
	return "Compartilhe este código:"
}
func (p *ptBR) InvitePanelNotice() string {
	return "válido por 10 minutos"
}
func (p *ptBR) InvitePanelGenerating() string {
	return "gerando convite…"
}
func (p *ptBR) InvitePanelWaiting() string {
	return "aguardando o contato aceitar…"
}
func (p *ptBR) InvitePanelError(err error) string {
	if err == nil {
		return "Falha ao consultar convites aceitos."
	}
	return "Falha ao consultar convites aceitos: " + err.Error()
}
func (p *ptBR) InstructionInvite() string {
	return "compartilhe o código · r novo · esc fechar"
}
func (p *ptBR) InstructionWaitingAPI() string {
	return "aguarde a resposta da API…"
}

// Nickname Panel
func (p *ptBR) NicknamePanelTitle() string { return "APELIDO DO CONTATO" }
func (p *ptBR) NicknamePanelPrompt(name string) string {
	return fmt.Sprintf("Novo apelido para %s (deixe vazio para remover):", name)
}
func (p *ptBR) NicknamePanelError(err error) string {
	if err == nil {
		return "Falha ao atualizar apelido."
	}
	return "Falha ao atualizar apelido: " + err.Error()
}
func (p *ptBR) ContactLabel() string {
	return "Contato: "
}
func (p *ptBR) NoNicknamePlaceholder() string {
	return "nenhum apelido (vazio)"
}
func (p *ptBR) InstructionNickname() string {
	return "digite o apelido · enter salvar · esc cancelar"
}

// Logs Panel
func (p *ptBR) LogsPanelTitle() string { return "LOGS DO SISTEMA" }
func (p *ptBR) LogsPanelSubtitle() string {
	return "Registro de eventos e conexões (ESC para fechar)"
}
func (p *ptBR) LogsPanelEmpty() string { return "Nenhum log registrado ainda." }

// Settings Panel
func (p *ptBR) SettingsPanelTitle() string       { return "CONFIGURAÇÕES" }
func (p *ptBR) SettingsLanguageSection() string  { return "Idioma / Language:" }
func (p *ptBR) InstructionSettings() string      { return "↑/↓ navegar  ·  enter aplicar  ·  esc fechar" }
