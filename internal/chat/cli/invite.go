package cli

import (
	"context"
	"fmt"

	cli "github.com/urfave/cli/v3"
)

func (a *App) InviteCommand(ctx context.Context, cmd *cli.Command) error {
	code, err := a.DiscoveryClient.CreateInvite(ctx, a.Identity.PeerID)
	if err != nil {
		return err
	}

	fmt.Println("Invite code:", code)
	return nil
}
