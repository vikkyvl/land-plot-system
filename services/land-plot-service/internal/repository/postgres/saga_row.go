package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const sagaColumns = "id, plot_id, kind, status, previous_state, new_version"

type plotStateRecord struct {
	Name       string        `json:"name"`
	Points     []pointRecord `json:"points"`
	AreaM2     float64       `json:"area_m2"`
	PerimeterM float64       `json:"perimeter_m"`
	Version    int64         `json:"version"`
}

func scanSaga(row rowScanner) (domain.Saga, error) {
	var (
		saga          domain.Saga
		previousState []byte
	)

	if err := row.Scan(
		&saga.ID, &saga.PlotID, &saga.Kind, &saga.Status, &previousState, &saga.NewVersion,
	); err != nil {
		return domain.Saga{}, err
	}

	state, err := decodePreviousState(previousState)
	if err != nil {
		return domain.Saga{}, err
	}

	saga.PreviousState = state

	return saga, nil
}

func encodePreviousState(state *domain.PlotState) ([]byte, error) {
	if state == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(plotStateRecord{
		Name:       state.Name,
		Points:     toPointRecords(state.Points),
		AreaM2:     state.AreaM2,
		PerimeterM: state.PerimeterM,
		Version:    state.Version,
	})
	if err != nil {
		return nil, fmt.Errorf(wrapEncodePreviousState, err)
	}

	return encoded, nil
}

func decodePreviousState(encoded []byte) (*domain.PlotState, error) {
	if encoded == nil {
		return nil, nil
	}

	var record plotStateRecord
	if err := json.Unmarshal(encoded, &record); err != nil {
		return nil, fmt.Errorf(wrapDecodePreviousState, err)
	}

	return &domain.PlotState{
		Name:       record.Name,
		Points:     toCoordinates(record.Points),
		AreaM2:     record.AreaM2,
		PerimeterM: record.PerimeterM,
		Version:    record.Version,
	}, nil
}
