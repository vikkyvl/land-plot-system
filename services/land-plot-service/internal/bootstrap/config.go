package bootstrap

import (
	"fmt"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func loadConfig() (config.Config, domain.AccuracyModel, error) {
	serviceConfig, err := config.Load()
	if err != nil {
		return config.Config{}, domain.AccuracyModel{}, fmt.Errorf(wrapLoadConfig, err)
	}

	accuracy, err := accuracyModel(serviceConfig.Accuracy)
	if err != nil {
		return config.Config{}, domain.AccuracyModel{}, err
	}

	return serviceConfig, accuracy, nil
}

func accuracyModel(settings config.AccuracyConfig) (domain.AccuracyModel, error) {
	fallback := domain.Capture{
		Zoom:    settings.FallbackZoom,
		Layer:   domain.MapLayer(settings.FallbackLayer),
		Pointer: domain.Pointer(settings.FallbackPointer),
	}
	if err := domain.ValidateCapture(&fallback); err != nil {
		return domain.AccuracyModel{}, fmt.Errorf(wrapFallbackCapture, err)
	}

	return domain.AccuracyModel{
		MouseClickPixels: settings.MouseClickPixels,
		TouchClickPixels: settings.TouchClickPixels,
		MapShiftM:        settings.MapShiftM,
		SatelliteShiftM:  settings.SatelliteShiftM,
		TreesM:           settings.TreesM,
		BuildingsM:       settings.BuildingsM,
		UnclearM:         settings.UnclearM,
		FallbackCapture:  fallback,
	}, nil
}

func newPlotService(
	repository service.PlotRepository,
	accuracy domain.AccuracyModel,
	sagaTimeout time.Duration,
	metrics service.PlotMetrics,
) *service.PlotService {
	return service.NewPlotService(repository, domain.LocalEllipsoidCalculator{}, accuracy, sagaTimeout, metrics)
}
