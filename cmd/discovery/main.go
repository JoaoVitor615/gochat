package discovery_main

import (
	"log"
	"net/http"

	"github.com/JoaoVitor615/gochat/internal/discovery"
)

func main() {
	store := discovery.NewStore("localhost:6379")
	defer store.Close()

	server := discovery.NewServer(store)

	log.Println("Discovery listening on :8080")

	if err := http.ListenAndServe(":8080", server.Routes()); err != nil {
		log.Fatal(err)
	}
}
