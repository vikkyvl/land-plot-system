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
	exitCode := bootstrap.RunMCPServer(runContext, os.Args[1:], os.Stderr)

	stop()
	os.Exit(exitCode)
}
