package balancer

import (
	"testing"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/discovery"
)

func TestRoundRobinSelect(t *testing.T) {
	t.Parallel()

	instances := []discovery.Instance{{ID: "one"}, {ID: "two"}, {ID: "three"}}
	balancer := NewRoundRobin()
	want := []string{"one", "two", "three", "one"}

	for index, expectedID := range want {
		selected, ok := balancer.Select("land-plot-service", instances)
		if !ok {
			t.Fatalf("selection %d unexpectedly failed", index)
		}

		if selected.ID != expectedID {
			t.Fatalf("selection %d = %q, want %q", index, selected.ID, expectedID)
		}
	}
}

func TestRoundRobinHasIndependentServiceCounters(t *testing.T) {
	t.Parallel()

	instances := []discovery.Instance{{ID: "one"}, {ID: "two"}}
	balancer := NewRoundRobin()
	balancer.Select("identity-service", instances)

	selected, ok := balancer.Select("report-service", instances)
	if !ok || selected.ID != "one" {
		t.Fatalf("first report selection = %+v, %v; want instance one", selected, ok)
	}
}
