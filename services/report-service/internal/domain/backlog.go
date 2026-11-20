package domain

import "time"

type OutboxBacklog struct {
	Pending   int64
	Failed    int64
	OldestAge time.Duration
}

type ReportCounts struct {
	ActivePlots  int64
	DeletedPlots int64
}
