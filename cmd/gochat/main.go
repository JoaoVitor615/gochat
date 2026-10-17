package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/JoaoVitor615/gochat/internal/chat/cli"
	"github.com/JoaoVitor615/gochat/internal/chat/logger"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	// Initialize logger to keep in-memory history and write to ~/.gochat/gochat.log without corrupting the TUI.
	_ = logger.Init()
	defer func() {
		_ = logger.Close()
	}()

	// Load .env.chat if present in current working directory.
	_ = godotenv.Load(".env.chat")

	// Also attempt to load .env.chat from the installed executable directory.
	if exePath, exeErr := os.Executable(); exeErr == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exePath), ".env.chat"))
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
