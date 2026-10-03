package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	cli "github.com/urfave/cli/v3"
)

func (a *App) InviteCommand(ctx context.Context, cmd *cli.Command) error {
	discoveryURL := os.Getenv("DISCOVERY_URL")
	if discoveryURL == "" {
		discoveryURL = "http://localhost:8080"
	}
	client := client.NewClient(discoveryURL)

	code, err := client.CreateInvite(ctx, a.Identity.PeerID)
	if err != nil {
		return err
	}

	fmt.Println("Invite code:", code)
	return nil
}
