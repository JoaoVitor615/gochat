package main

import (
	"context"
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/cli"
)

func main() {
	app := cli.InitApp()

	cmd := cli.InitCli(app)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}
