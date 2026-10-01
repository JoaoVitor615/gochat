package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/JoaoVitor615/gochat/internal/discovery"
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

	server := discovery.NewServer(service.New(redisRepository))

	log.Printf("Discovery listening on :%s", port)

	if err := http.ListenAndServe(":"+port, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
