package service

import (
	"context"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type TerrainService struct {
	elevations ElevationProvider
	settings   domain.TerrainSettings
}

func NewTerrainService(elevations ElevationProvider, settings domain.TerrainSettings) *TerrainService {
	return &TerrainService{elevations: elevations, settings: settings}
}

func (terrainService *TerrainService) Profile(
	requestContext context.Context,
	boundary []domain.Coordinate,
	areaM2 float64,
) (domain.Terrain, error) {
	grid := terrainService.settings.Grid(boundary)

	elevations, err := terrainService.elevations.Elevations(requestContext, grid)
	if err != nil {
		return domain.UnavailableTerrain(), fmt.Errorf(wrapGetElevations, err)
	}

	terrain, err := terrainService.settings.Profile(grid, elevations, areaM2)
	if err != nil {
		return domain.UnavailableTerrain(), fmt.Errorf(wrapProfileTerrain, err)
	}

	return terrain, nil
}
