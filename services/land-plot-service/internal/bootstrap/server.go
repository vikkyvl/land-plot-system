package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

const readHeaderTimeout = 5 * time.Second

func serve(
	runContext context.Context,
	server *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	beforeShutdown func(),
	logger *slog.Logger,
) error {
	serveErrors := make(chan error, 1)

	go func() {
		logger.Info(logHTTPServerStarted, logging.KeyAddress, listener.Addr().String())

		serveErrors <- server.Serve(listener)
	}()

	select {
	case err := <-serveErrors:
		beforeShutdown()
		return fmt.Errorf(wrapServeHTTP, err)
	case <-runContext.Done():
		logger.Info(logShutdownRequested)
	}

	beforeShutdown()

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf(wrapShutdownHTTPServer, err)
	}

	if err := <-serveErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf(wrapServeHTTP, err)
	}

	logger.Info(logHTTPServerStopped)

	return nil
}
