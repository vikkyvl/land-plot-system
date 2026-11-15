package bootstrap

import (
	"context"
	"sync"
)

func startWorkers(runContext context.Context, workers ...func(context.Context)) (wait func()) {
	var group sync.WaitGroup

	for _, worker := range workers {
		group.Go(func() { worker(runContext) })
	}

	return group.Wait
}
