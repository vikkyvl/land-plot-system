package balancer

import (
	"sync"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/discovery"
)

type RoundRobin struct {
	mutex sync.Mutex
	next  map[string]uint64
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{next: make(map[string]uint64)}
}

func (roundRobin *RoundRobin) Select(service string, instances []discovery.Instance) (discovery.Instance, bool) {
	if len(instances) == 0 {
		return discovery.Instance{}, false
	}

	roundRobin.mutex.Lock()
	defer roundRobin.mutex.Unlock()

	index := roundRobin.next[service] % uint64(len(instances))
	roundRobin.next[service]++

	return instances[index], true
}
