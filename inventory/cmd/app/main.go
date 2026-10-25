package main

import (
	"log/slog"
	"os"

	"rent/internal/platform"
	"rent/inventory/internal/app"
)

func main() {
	ctx, cancel := platform.Context()
	defer cancel()
	if err := app.Run(ctx); err != nil {
		slog.Error("service stopped", "service", "inventory", "error", err)
		os.Exit(1)
	}
}
