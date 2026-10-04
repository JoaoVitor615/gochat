package cli

import (
	"context"

	"github.com/JoaoVitor615/gochat/internal/chat/tui"
	cli "github.com/urfave/cli/v3"
)

func (a *App) GoChatCommand(_ context.Context, _ *cli.Command) error {
	return tui.Run()
}
