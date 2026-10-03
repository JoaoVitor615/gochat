package cli

import (
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
)

type App struct {
	Identity        *identity.Identity
	DiscoveryClient *client.Client
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
