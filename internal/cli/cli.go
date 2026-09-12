package cli

import (
	"context"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/identity"
	"github.com/JoaoVitor615/gochat/internal/p2p"
	cli "github.com/urfave/cli/v3"
)

func InitCli() *cli.Command {
	return &cli.Command{
		Name:  "gochat",
		Usage: "P2P chat from your terminal",
		Action: func(context.Context, *cli.Command) error {
			fmt.Println("Initializing gochat...")

			id, err := identity.NewIdentity()
			if err != nil {
				return err
			}

			host, err := p2p.NewHost(id, 50)
			if err != nil {
				return err
			}

			defer host.Close()

			fmt.Println("GoChat iniciado!")
			fmt.Println()
			fmt.Println("Peer ID:", host.ID())
			fmt.Println()
			fmt.Println("Escutando em:")

			for _, addr := range host.Addrs() {
				fmt.Println(" ", addr)
			}

			return nil
		},

		Commands: []*cli.Command{
			{
				Name:  "invite",
				Usage: "create an invite",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					println("Creating invite...")
					return nil
				},
			},
			{
				Name:  "add",
				Usage: "add a peer",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					println("Adding peer:", cmd.Args().First())
					return nil
				},
			},
			{
				Name:  "friends",
				Usage: "list your friends",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					println("Friends:")
					return nil
				},
			},
			{
				Name:  "chat",
				Usage: "open a chat",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					println("Chat with:", cmd.Args().First())
					return nil
				},
			},
		},
	}

}
