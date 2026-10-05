package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/JoaoVitor615/gochat/internal/discovery"
	"github.com/JoaoVitor615/gochat/internal/discovery/observation"
	"github.com/JoaoVitor615/gochat/internal/discovery/repository"
	"github.com/JoaoVitor615/gochat/internal/discovery/service"
)

func main() {
	fmt.Println("Hello, World!")
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	port := os.Getenv("DISCOVERY_PORT")
	if port == "" {
		port = "8080"
	}

	redisRepository := repository.NewRedis(redisAddr)
	defer redisRepository.Close()

	addressObserver, err := observation.New(os.Getenv("DISCOVERY_OBSERVER_ADVERTISE_ADDR"), "/var/lib/gochat-observer")
	if err != nil {
		log.Fatal(err)
	}
	defer addressObserver.Close()
	if addressObserver.Enabled() {
		log.Printf("libp2p address observer listening on UDP %d", observation.DefaultListenPort)
		if observerAddress, err := addressObserver.Address(); err != nil {
			log.Printf("could not determine observer address: %v", err)
		} else {
			log.Printf("libp2p address observer ready at %s", observerAddress)
		}
	} else {
		log.Printf("libp2p address observer disabled: DISCOVERY_OBSERVER_ADVERTISE_ADDR is empty")
	}

	server := discovery.NewServer(service.New(redisRepository), addressObserver)

	log.Printf("Discovery listening on :%s", port)

	if err := http.ListenAndServe(":"+port, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
