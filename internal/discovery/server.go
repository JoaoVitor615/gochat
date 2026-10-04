package discovery

import (
	"net/http"

	"github.com/JoaoVitor615/gochat/internal/discovery/handler"
	"github.com/JoaoVitor615/gochat/internal/discovery/service"
)

type Server struct {
	handler *handler.Handler
}

func NewServer(discoveryService *service.Service) *Server {
	return &Server{
		handler: handler.New(discoveryService),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/heartbeat", s.handler.Heartbeat)
	mux.HandleFunc("/invite", s.handler.Invite)
	mux.HandleFunc("/resolve", s.handler.Resolve)
	mux.HandleFunc("/", s.handler.NotFound)

	return mux
}
