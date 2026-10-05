package cli

import (
	"context"

	"github.com/JoaoVitor615/gochat/internal/chat/tui"
)

func (a *App) GoChatCommand(ctx context.Context) error {
	stopPresence, err := a.startPresence(ctx)
	if err != nil {
		return err
	}
	defer stopPresence()

	return tui.Run(ctx, a.Identity.PeerID, a.DiscoveryClient)
}
