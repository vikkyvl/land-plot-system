package domain

import (
	"strings"
	"time"
)

const MaximumNameLength = 120

type PlotStatus string

const (
	StatusPending   PlotStatus = "PENDING"
	StatusConfirmed PlotStatus = "CONFIRMED"
	StatusCancelled PlotStatus = "CANCELLED"
)

type Plot struct {
	ID         string
	OwnerID    string
	Name       string
	Points     []Coordinate
	AreaM2     float64
	PerimeterM float64
	Version    int64
	Status     PlotStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PlotDraft struct {
	OwnerID string
	Name    string
	Points  []Coordinate
}

type PlotChanges struct {
	Name   *string
	Points *[]Coordinate
}

type PlotState struct {
	Name       string
	Points     []Coordinate
	AreaM2     float64
	PerimeterM float64
	Version    int64
}

func NewPlot(id string, draft PlotDraft, calculator AreaCalculator, now time.Time) (Plot, error) {
	normalizedName, err := normalizeName(draft.Name)
	if err != nil {
		return Plot{}, err
	}

	result, err := measure(draft.Points, calculator)
	if err != nil {
		return Plot{}, err
	}

	createdAt := now.UTC()

	return Plot{
		ID:         id,
		OwnerID:    draft.OwnerID,
		Name:       normalizedName,
		Points:     clonePoints(draft.Points),
		AreaM2:     result.AreaM2,
		PerimeterM: result.PerimeterM,
		Version:    1,
		Status:     StatusPending,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}, nil
}

func (plot Plot) Updated(changes PlotChanges, calculator AreaCalculator, now time.Time) (Plot, error) {
	if changes.Name == nil && changes.Points == nil {
		return Plot{}, ErrNoChanges
	}

	if plot.Status != StatusConfirmed {
		return Plot{}, ErrPlotNotConfirmed
	}

	updated := plot

	if changes.Name != nil {
		name, err := normalizeName(*changes.Name)
		if err != nil {
			return Plot{}, err
		}

		updated.Name = name
	}

	if changes.Points != nil {
		result, err := measure(*changes.Points, calculator)
		if err != nil {
			return Plot{}, err
		}

		updated.Points = clonePoints(*changes.Points)
		updated.AreaM2 = result.AreaM2
		updated.PerimeterM = result.PerimeterM
	}

	updated.Version++
	updated.Status = StatusPending
	updated.UpdatedAt = now.UTC()

	return updated, nil
}

func (plot Plot) State() PlotState {
	return PlotState{
		Name:       plot.Name,
		Points:     clonePoints(plot.Points),
		AreaM2:     plot.AreaM2,
		PerimeterM: plot.PerimeterM,
		Version:    plot.Version,
	}
}

func (plot Plot) Confirmed() Plot {
	plot.Status = StatusConfirmed

	return plot
}

func (plot Plot) Cancelled(now time.Time) Plot {
	plot.Status = StatusCancelled
	plot.Version++
	plot.UpdatedAt = now.UTC()

	return plot
}

func (plot Plot) Restored(previous PlotState, now time.Time) Plot {
	plot.Name = previous.Name
	plot.Points = clonePoints(previous.Points)
	plot.AreaM2 = previous.AreaM2
	plot.PerimeterM = previous.PerimeterM
	plot.Status = StatusConfirmed
	plot.Version++
	plot.UpdatedAt = now.UTC()

	return plot
}

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > MaximumNameLength {
		return "", ErrInvalidName
	}

	return name, nil
}

func clonePoints(points []Coordinate) []Coordinate {
	return append([]Coordinate(nil), points...)
}
