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
	Survey     Survey
	Accuracy   Accuracy
	Version    int64
	Status     PlotStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type PlotDraft struct {
	OwnerID string
	Name    string
	Points  []Coordinate
	Survey  Survey
}

type PlotChanges struct {
	Name         *string
	Points       *[]Coordinate
	Capture      *Capture
	Obstructions *[]Obstruction
}

type PlotState struct {
	Name       string
	Points     []Coordinate
	AreaM2     float64
	PerimeterM float64
	Survey     Survey
	Version    int64
}

func NewPlot(id string, draft PlotDraft, calculator AreaCalculator, now time.Time) (Plot, error) {
	normalizedName, err := normalizeName(draft.Name)
	if err != nil {
		return Plot{}, err
	}

	if err := draft.Survey.Validate(len(draft.Points)); err != nil {
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
		Survey:     draft.Survey.clone(),
		Version:    1,
		Status:     StatusPending,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}, nil
}

func (plot Plot) Updated(changes PlotChanges, calculator AreaCalculator, now time.Time) (Plot, error) {
	if changes.Capture != nil && changes.Points == nil {
		return Plot{}, ErrInvalidCapture
	}

	if changes.empty() {
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

	survey := changes.survey(plot.Survey)
	if err := survey.Validate(len(changes.pointsOr(plot.Points))); err != nil {
		return Plot{}, err
	}

	updated.Survey = survey

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

func (plot Plot) Deleted(now time.Time) (Plot, error) {
	if plot.Status != StatusConfirmed {
		return Plot{}, ErrPlotNotConfirmed
	}

	deletedAt := now.UTC()

	deleted := plot
	deleted.Version++
	deleted.UpdatedAt = deletedAt
	deleted.DeletedAt = &deletedAt

	return deleted, nil
}

func (plot Plot) IsDeleted() bool {
	return plot.DeletedAt != nil
}

func (plot Plot) State() PlotState {
	return PlotState{
		Name:       plot.Name,
		Points:     clonePoints(plot.Points),
		AreaM2:     plot.AreaM2,
		PerimeterM: plot.PerimeterM,
		Survey:     plot.Survey.clone(),
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
	plot.Survey = previous.Survey.clone()
	plot.Status = StatusConfirmed
	plot.Version++
	plot.UpdatedAt = now.UTC()

	return plot
}

func (changes PlotChanges) empty() bool {
	return changes.Name == nil && changes.Points == nil && changes.Obstructions == nil
}

func (changes PlotChanges) pointsOr(current []Coordinate) []Coordinate {
	if changes.Points != nil {
		return *changes.Points
	}

	return current
}

func (changes PlotChanges) survey(current Survey) Survey {
	survey := current.clone()
	if changes.Points != nil {
		survey = Survey{Capture: changes.Capture}
	}

	if changes.Obstructions != nil {
		survey.Obstructions = append([]Obstruction(nil), *changes.Obstructions...)
	}

	return survey
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
