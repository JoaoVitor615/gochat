package cli

import (
	"context"
	"log"

	"github.com/JoaoVitor615/gochat/internal/chat/tui"
)

func (a *App) GoChatCommand(ctx context.Context) error {
	stopPresence, err := a.startPresence(ctx)
	if err != nil {
		return err
	}
	defer stopPresence()
	if a.Messenger != nil {
		go func() {
			if err := a.Messenger.RetryAll(ctx); err != nil && ctx.Err() == nil {
				log.Printf("retry queued messages at startup: %v", err)
			}
		}()
	}

	return tui.Run(ctx, a.Identity.PeerID, a.DiscoveryClient, a.Store, a.Messenger)
}
