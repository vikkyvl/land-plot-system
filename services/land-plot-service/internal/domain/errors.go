package domain

import "fmt"

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
)

var (
	ErrNoChanges = NewError(KindInvalidRequest, "no_changes",
		"at least one field must be provided")

	ErrInvalidPlotID = NewError(KindInvalidRequest, "invalid_plot_id",
		"plot ID must be a valid UUID")

	ErrPlotNotFound = NewError(KindNotFound, "plot_not_found",
		"plot not found")

	ErrForbidden = NewError(KindForbidden, "forbidden",
		"plot belongs to another user")

	ErrVersionConflict = NewError(KindConflict, "version_conflict",
		"plot was modified concurrently; retry the request")
)
