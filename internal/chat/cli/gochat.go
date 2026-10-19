package cli

import (
	"context"
	"log"

	"github.com/JoaoVitor615/gochat/internal/chat/tui"
)

func (a *App) GoChatCommand(ctx context.Context) error {
	var stopPresence func()
	defer func() {
		if stopPresence != nil {
			stopPresence()
		}
	}()

	return tui.RunWithSplash(ctx, a.Identity.PeerID, a.DiscoveryClient, a.Store, a.Bus, func(initCtx context.Context) (tui.MessageSender, func(), error) {
		stop, err := a.startPresence(initCtx)
		if err != nil {
			return nil, nil, err
		}
		stopPresence = stop
		if a.Messenger != nil {
			go func() {
				if err := a.Messenger.RetryAll(initCtx); err != nil && initCtx.Err() == nil {
					log.Printf("retry queued messages at startup: %v", err)
				}
			}()
		}
		return a.Messenger, stop, nil
	})
}
