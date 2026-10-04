package main

import (
	"context"
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/cli"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env.chat"); err != nil {
		log.Fatalf("load .env.chat: %v", err)
	}

	app := cli.InitApp()

	cmd := cli.InitCli(app)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}
