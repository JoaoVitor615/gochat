package cli

import (
	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
)

type App struct {
	Identity        *identity.Identity
	DiscoveryClient *client.Client
}
