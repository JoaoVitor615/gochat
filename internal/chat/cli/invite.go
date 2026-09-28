package cli

import (
	"context"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	cli "github.com/urfave/cli/v3"
)

func InviteCommand(ctx context.Context, cmd *cli.Command) error {
	id, err := identity.NewIdentity()
	if err != nil {
		return err
	}

	client := client.NewClient("")

	code, err := client.CreateInvite(id.GetPeerID())
	if err != nil {
		return err
	}

	fmt.Println("Invite code:", code)
	return nil
}
