package main

import (
	"log/slog"
	"os"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/bootstrap"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		slog.Error(logServiceStopped, logging.KeyError, err)
		os.Exit(1)
	}
}
