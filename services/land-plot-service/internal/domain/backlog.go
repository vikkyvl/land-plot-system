package domain

import "time"

type OutboxBacklog struct {
	Pending   int64
	Failed    int64
	OldestAge time.Duration
}

type PlotCounts struct {
	Pending   int64
	Confirmed int64
	Cancelled int64
	Deleted   int64
}
