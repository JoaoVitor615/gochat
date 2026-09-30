package cli

import (
	"context"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	cli "github.com/urfave/cli/v3"
)

func (a *App) GoChatCommand(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Initializing gochat...")

	port := cmd.Int(FLAG_PORT)

	host, err := p2p.NewHost(a.Identity, port)
	if err != nil {
		return err
	}

	defer host.Close()

	fmt.Println("GoChat initialized!")
	fmt.Println("Peer ID:", a.Identity.PeerID)
	fmt.Println("Listening on:")

	for _, addr := range host.Addrs() {
		fmt.Println(" ", addr)
	}

	p2p.SetStreamHandler(host, p2p.HandleStream)

	<-ctx.Done()

	return nil
}
