package domain

import (
	"errors"
	"fmt"
)

var ErrMessageRejected = errors.New("message rejected by the broker")

var (
	ErrInvalidName = NewError(KindValidation, "invalid_name",
		fmt.Sprintf("plot name must contain between 1 and %d characters", MaximumNameLength))

	ErrInsufficientPoints = NewError(KindValidation, "insufficient_points",
		fmt.Sprintf("polygon must contain at least %d points", MinimumPolygonPoints))

	ErrInvalidCoordinate = NewError(KindValidation, "invalid_coordinate",
		fmt.Sprintf("latitude must be between %d and %d and longitude between %d and %d",
			minimumLatitude, maximumLatitude, minimumLongitude, maximumLongitude))

	ErrDuplicatePoint = NewError(KindValidation, "duplicate_point",
		"polygon points must be distinct")

	ErrSelfIntersection = NewError(KindValidation, "self_intersection",
		"polygon must not intersect itself")

	ErrZeroArea = NewError(KindValidation, "zero_area",
		"polygon area must be positive")

	ErrTooManyPoints = NewError(KindValidation, "too_many_points",
		fmt.Sprintf("polygon must contain at most %d points", MaximumPolygonPoints))
)

var (
	ErrNoChanges = NewError(KindInvalidRequest, "no_changes",
		"at least one field must be provided")

	ErrInvalidCapture = NewError(
		KindInvalidRequest,
		"invalid_capture",
		fmt.Sprintf(
			"capture needs a zoom between %g and %g, a layer map or satellite and a pointer mouse, touch or pen",
			MinimumZoom,
			MaximumZoom,
		),
	)

	ErrInvalidObstruction = NewError(KindInvalidRequest, "invalid_obstruction",
		"obstruction needs an existing side, marked once, and a reason trees, buildings or unclear")

	ErrInvalidPlotID = NewError(KindInvalidRequest, "invalid_plot_id",
		"plot ID must be a valid UUID")

	ErrPlotNotFound = NewError(KindNotFound, "plot_not_found",
		"plot not found")

	ErrForbidden = NewError(KindForbidden, "forbidden",
		"plot belongs to another user")

	ErrVersionConflict = NewError(KindConflict, "version_conflict",
		"plot was modified concurrently; retry the request")

	ErrPlotNotConfirmed = NewError(KindConflict, "plot_not_confirmed",
		"plot is waiting for confirmation; retry the request later")

	ErrPlotDeleted = NewError(KindGone, "plot_deleted",
		"plot was deleted")
)
