package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/vleshchenko/land-plot-system/ai/internal/bootstrap"
)

func main() {
	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	exitCode := bootstrap.RunProducerAgent(runContext, os.Stderr)

	stop()
	os.Exit(exitCode)
}
