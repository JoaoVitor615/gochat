package discovery

import (
	"net/http"

	"github.com/JoaoVitor615/gochat/internal/discovery/handler"
	"github.com/JoaoVitor615/gochat/internal/discovery/observation"
	"github.com/JoaoVitor615/gochat/internal/discovery/service"
)

type Server struct {
	handler *handler.Handler
}

func NewServer(discoveryService *service.Service, addressObserver *observation.Observer) *Server {
	return &Server{
		handler: handler.New(discoveryService, addressObserver),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/heartbeat", s.handler.Heartbeat)
	mux.HandleFunc("/invite", s.handler.Invite)
	mux.HandleFunc("/resolve", s.handler.Resolve)
	mux.HandleFunc("/observer", s.handler.Observer)
	mux.HandleFunc("/", s.handler.NotFound)

	return mux
}
