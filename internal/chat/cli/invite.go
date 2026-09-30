package cli

import (
	"context"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	cli "github.com/urfave/cli/v3"
)

func (a *App) InviteCommand(ctx context.Context, cmd *cli.Command) error {
	client := client.NewClient("http://localhost:8080")

	code, err := client.CreateInvite(ctx, a.Identity.PeerID)
	if err != nil {
		return err
	}

	fmt.Println("Invite code:", code)
	return nil
}
