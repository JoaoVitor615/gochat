package cli

import (
	"context"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/identity"
	"github.com/JoaoVitor615/gochat/internal/p2p"
	cli "github.com/urfave/cli/v3"
)

func GoChatCommand(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Initializing gochat...")

	id, err := identity.NewIdentity()
	if err != nil {
		return err
	}

	port := cmd.Int(FLAG_PORT)

	host, err := p2p.NewHost(id, port)
	if err != nil {
		return err
	}

	defer host.Close()

	fmt.Println("GoChat initialized!")
	fmt.Println("Peer ID:", host.ID())
	fmt.Println("Listening on:")

	for _, addr := range host.Addrs() {
		fmt.Println(" ", addr)
	}

	<-ctx.Done()

	return nil
}
