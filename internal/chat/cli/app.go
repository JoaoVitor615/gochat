package cli

import (
	"fmt"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	"github.com/JoaoVitor615/gochat/internal/chat/messaging"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	"github.com/libp2p/go-libp2p/core/host"
)

type App struct {
	Identity        *identity.Identity
	DiscoveryClient *client.Client
	Messenger       *messaging.Service
	Store           storage.Repository
	Host            host.Host
}

func InitApp() (*App, error) {
	id, err := identity.NewIdentity()
	if err != nil {
		return nil, fmt.Errorf("initialize identity: %w", err)
	}

	discoveryURL := os.Getenv("DISCOVERY_URL")
	client, err := client.NewClient(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("initialize discovery client: %w", err)
	}

	store, err := storage.OpenDefault()
	if err != nil {
		return nil, fmt.Errorf("initialize local storage: %w", err)
	}

	return &App{
		Identity:        id,
		DiscoveryClient: client,
		Store:           store,
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.Store == nil {
		return nil
	}
	return a.Store.Close()
}
