package discovery_main

import (
	"log"
	"net/http"

	"github.com/JoaoVitor615/gochat/internal/discovery"
	"github.com/JoaoVitor615/gochat/internal/discovery/repository"
	"github.com/JoaoVitor615/gochat/internal/discovery/service"
)

func main() {
	redisRepository := repository.NewRedis("localhost:6379")
	defer redisRepository.Close()

	server := discovery.NewServer(service.New(redisRepository))

	log.Println("Discovery listening on :8080")

	if err := http.ListenAndServe(":8080", server.Routes()); err != nil {
		log.Fatal(err)
	}
}
