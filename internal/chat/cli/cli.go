package cli

import (
	"context"

	cli "github.com/urfave/cli/v3"
)

func InitCli(app *App) *cli.Command {
	return &cli.Command{
		Name:  "gochat",
		Usage: "P2P chat from your terminal",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return app.GoChatCommand(ctx)
		},
		Commands: []*cli.Command{
			{
				Name:  "invite",
				Usage: "create an invite",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return app.InviteCommand(ctx, cmd)
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
