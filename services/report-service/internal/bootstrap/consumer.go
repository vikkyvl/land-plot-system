package bootstrap

import (
	"context"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/messaging/rabbitmq"
)

func startConsumer(
	runContext context.Context,
	consumer *rabbitmq.Consumer,
	handler rabbitmq.EventHandler,
	stopService context.CancelFunc,
) (wait func() error) {
	done := make(chan error, 1)

	go func() {
		err := consumer.Run(runContext, handler)

		stopService()

		if err != nil {
			err = fmt.Errorf(wrapConsumePlotEvents, err)
		}

		done <- err
	}()

	return func() error { return <-done }
}
