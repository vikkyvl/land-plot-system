package service

import "github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"

type metricsSpy struct {
	created   []string
	downloads []string
	published int
	failures  []bool
}

func (spy *metricsSpy) ReportCreated(terrain, climate domain.DataStatus) {
	spy.created = append(spy.created, string(terrain)+" "+string(climate))
}

func (spy *metricsSpy) ReportDownloaded(extension string) {
	spy.downloads = append(spy.downloads, extension)
}

func (spy *metricsSpy) OutboxPublished(count int) {
	spy.published += count
}

func (spy *metricsSpy) OutboxPublishFailed(abandoned bool) {
	spy.failures = append(spy.failures, abandoned)
}
