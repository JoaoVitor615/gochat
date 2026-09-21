package main

import (
	"context"
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/cli"
)

func main() {
	cmd := cli.InitCli()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}
