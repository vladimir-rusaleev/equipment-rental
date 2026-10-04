package main

import (
	"log/slog"
	"os"

	"rent/internal/platform"
	"rent/rental/internal/app"
)

func main() {
	ctx, cancel := platform.Context()
	defer cancel()
	if err := app.Run(ctx); err != nil {
		slog.Error("service stopped", "service", "rental", "error", err)
		os.Exit(1)
	}
}
