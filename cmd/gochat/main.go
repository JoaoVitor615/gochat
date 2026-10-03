package main

import (
	"context"
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/cli"
	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
)

func main() {
	discoveryURL := os.Getenv("DISCOVERY_URL")

	id, err := identity.NewIdentity()
	if err != nil {
		log.Fatal(err)
	}

	client, err := client.NewClient(discoveryURL)
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Identity:        id,
		DiscoveryClient: client,
	}

	cmd := cli.InitCli(app)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}
