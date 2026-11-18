package domain

const (
	DefaultReportLimit = 50
	MaximumReportLimit = 100
)

type ReportFilter struct {
	OwnerID string
	PlotID  string
	Limit   int
}
