package cli

import (
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	"github.com/libp2p/go-libp2p/core/host"
)

type App struct {
	Identity        *identity.Identity
	DiscoveryClient *client.Client
	Host            host.Host
}

func InitApp() *App {
	id, err := identity.NewIdentity()
	if err != nil {
		log.Fatal(err)
	}

	discoveryURL := os.Getenv("DISCOVERY_URL")

	client, err := client.NewClient(discoveryURL)
	if err != nil {
		log.Fatal(err)
	}

	return &App{
		Identity:        id,
		DiscoveryClient: client,
	}
}
