package cli

import (
	"context"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	cli "github.com/urfave/cli/v3"
)

func (a *App) GoChatCommand(ctx context.Context, cmd *cli.Command) error {
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
	fmt.Println("Peer ID:", id.PeerID)
	fmt.Println("Listening on:")

	for _, addr := range host.Addrs() {
		fmt.Println(" ", addr)
	}

	p2p.SetStreamHandler(host, p2p.HandleStream)

	<-ctx.Done()

	return nil
}
