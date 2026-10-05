package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/JoaoVitor615/gochat/internal/chat/cli"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	if err := godotenv.Load(".env.chat"); err != nil {
		return fmt.Errorf("load .env.chat: %w", err)
	}

	app, err := cli.InitApp()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close app: %w", closeErr))
		}
	}()

	cmd := cli.InitCli(app)

	return cmd.Run(context.Background(), os.Args)
}
