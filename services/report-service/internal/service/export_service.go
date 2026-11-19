package service

import (
	"context"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type ExportService struct {
	reports   ReportReader
	documents ReportDocuments
}

func NewExportService(reports ReportReader, documents ReportDocuments) *ExportService {
	return &ExportService{reports: reports, documents: documents}
}

func (exportService *ExportService) PDF(
	requestContext context.Context,
	ownerID, reportID, languageCode string,
) (domain.ReportFile, error) {
	language, err := domain.ParseLanguage(languageCode)
	if err != nil {
		return domain.ReportFile{}, err
	}

	report, err := exportService.reports.Get(requestContext, ownerID, reportID)
	if err != nil {
		return domain.ReportFile{}, err
	}

	content, err := exportService.documents.PDF(report, language)
	if err != nil {
		return domain.ReportFile{}, fmt.Errorf(wrapRenderPDF, err)
	}

	return domain.ReportFile{
		Name:        report.FileName(language.ReportFilePrefix(), domain.ExtensionPDF),
		ContentType: domain.ContentTypePDF,
		Content:     content,
	}, nil
}

func (exportService *ExportService) GeoJSON(
	requestContext context.Context,
	ownerID, reportID, languageCode string,
) (domain.ReportFile, error) {
	language, err := domain.ParseLanguage(languageCode)
	if err != nil {
		return domain.ReportFile{}, err
	}

	report, err := exportService.reports.Get(requestContext, ownerID, reportID)
	if err != nil {
		return domain.ReportFile{}, err
	}

	content, err := exportService.documents.GeoJSON(report)
	if err != nil {
		return domain.ReportFile{}, err
	}

	return domain.ReportFile{
		Name:        report.FileName(language.ReportFilePrefix(), domain.ExtensionGeoJSON),
		ContentType: domain.ContentTypeGeoJSON,
		Content:     content,
	}, nil
}
