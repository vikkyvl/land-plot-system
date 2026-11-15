package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

type pointArgument struct {
	Latitude  *float64 `json:"lat"`
	Longitude *float64 `json:"lon"`
}

type listPlotsArguments struct {
	NameContains string `json:"name_contains"`
}

type plotIDArguments struct {
	PlotID string `json:"plot_id"`
}

type reportIDArguments struct {
	ReportID string `json:"report_id"`
}

type createPlotArguments struct {
	Name   *string         `json:"name"`
	Points []pointArgument `json:"points"`
}

type updatePlotArguments struct {
	PlotID string          `json:"plot_id"`
	Name   *string         `json:"name"`
	Points []pointArgument `json:"points"`
}

type waitPlotArguments struct {
	PlotID          string `json:"plot_id"`
	ExpectedVersion *int64 `json:"expected_version"`
	TimeoutSeconds  *int   `json:"timeout_seconds"`
}

func decodeArguments[Arguments any](raw json.RawMessage) (Arguments, error) {
	var arguments Arguments

	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage("{}")
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&arguments); err != nil {
		return arguments, fmt.Errorf(wrapDecodeArguments, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return arguments, errTrailingArguments
	}

	return arguments, nil
}

func validateUUID(field, value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf(formatMustBeUUID, field)
	}

	return nil
}

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maximumNameLength {
		return "", errInvalidNameLength
	}

	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", errNameControlCharacters
	}

	return name, nil
}

func toCoordinates(points []pointArgument) ([]landapi.Coordinate, error) {
	if len(points) < minimumPolygonPoints || len(points) > maximumPolygonPoints {
		return nil, errInvalidPointCount
	}

	coordinates := make([]landapi.Coordinate, len(points))
	for i, point := range points {
		if point.Latitude == nil || point.Longitude == nil {
			return nil, fmt.Errorf(formatPointMissingCoordinate, i+1)
		}

		latitude, longitude := *point.Latitude, *point.Longitude
		if latitude < minimumLatitude || latitude > maximumLatitude ||
			longitude < minimumLongitude || longitude > maximumLongitude {
			return nil, fmt.Errorf(formatPointOutOfRange, i+1)
		}

		coordinates[i] = landapi.Coordinate{Latitude: latitude, Longitude: longitude}
	}

	if coordinates[0] == coordinates[len(coordinates)-1] {
		return nil, errClosedPolygon
	}

	return coordinates, nil
}

func (arguments createPlotArguments) draft() (landapi.PlotDraft, error) {
	if arguments.Name == nil {
		return landapi.PlotDraft{}, fmt.Errorf(formatMissingArgument, "name")
	}

	name, err := normalizeName(*arguments.Name)
	if err != nil {
		return landapi.PlotDraft{}, err
	}

	points, err := toCoordinates(arguments.Points)
	if err != nil {
		return landapi.PlotDraft{}, err
	}

	return landapi.PlotDraft{Name: name, Points: points}, nil
}

func (arguments updatePlotArguments) changes() (landapi.PlotChanges, error) {
	if err := validateUUID("plot_id", arguments.PlotID); err != nil {
		return landapi.PlotChanges{}, err
	}

	if arguments.Name == nil && arguments.Points == nil {
		return landapi.PlotChanges{}, errNoChanges
	}

	changes := landapi.PlotChanges{}

	if arguments.Name != nil {
		name, err := normalizeName(*arguments.Name)
		if err != nil {
			return landapi.PlotChanges{}, err
		}

		changes.Name = &name
	}

	if arguments.Points != nil {
		points, err := toCoordinates(arguments.Points)
		if err != nil {
			return landapi.PlotChanges{}, err
		}

		changes.Points = &points
	}

	return changes, nil
}

func (arguments waitPlotArguments) validate() error {
	if err := validateUUID("plot_id", arguments.PlotID); err != nil {
		return err
	}

	if arguments.ExpectedVersion == nil || *arguments.ExpectedVersion < minimumVersion {
		return errInvalidExpectedVersion
	}

	if arguments.TimeoutSeconds == nil || *arguments.TimeoutSeconds < minimumTimeoutSeconds ||
		*arguments.TimeoutSeconds > maximumTimeoutSeconds {
		return errInvalidTimeout
	}

	return nil
}
