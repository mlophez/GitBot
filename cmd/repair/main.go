package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	// Get all apps
	// Get info of pull request
	// if open, check target branch go to pull request
	// if not pull request check target branch is in default branch
}
