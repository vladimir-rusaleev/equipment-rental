package main

import (
	"log/slog"
	"os"

	"rent/gateway/internal/app"
	"rent/internal/platform"
)

func main() {
	ctx, cancel := platform.Context()
	defer cancel()
	if err := app.Run(ctx); err != nil {
		slog.Error("service stopped", "service", "gateway", "error", err)
		os.Exit(1)
	}
}
