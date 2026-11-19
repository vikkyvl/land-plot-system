package service

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type ReportQuery struct {
	PlotID string
	Limit  string
}

func newReportFilter(ownerID string, query ReportQuery) (domain.ReportFilter, error) {
	filter := domain.ReportFilter{OwnerID: ownerID, PlotID: query.PlotID, Limit: domain.DefaultReportLimit}

	if query.PlotID != "" {
		if _, err := uuid.Parse(query.PlotID); err != nil {
			return domain.ReportFilter{}, domain.ErrInvalidQuery
		}
	}

	if query.Limit != "" {
		limit, err := strconv.Atoi(query.Limit)
		if err != nil || limit < 1 || limit > domain.MaximumReportLimit {
			return domain.ReportFilter{}, domain.ErrInvalidQuery
		}

		filter.Limit = limit
	}

	return filter, nil
}
