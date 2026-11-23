package tools

import (
	"errors"
	"fmt"
)

var (
	errTrailingArguments     = errors.New("arguments must contain exactly one JSON object")
	errInvalidNameLength     = fmt.Errorf("name must contain from 1 to %d characters", maximumNameLength)
	errNameControlCharacters = errors.New("name must not contain control characters")
	errInvalidPointCount     = fmt.Errorf("points must contain from %d to %d points",
		minimumPolygonPoints, maximumPolygonPoints)
	errClosedPolygon = errors.New("the polygon is open: do not repeat the first point at the end")
	errNoChanges     = errors.New("pass at least one of name or points")

	errInvalidExpectedVersion = fmt.Errorf("expected_version must be an integer of at least %d", minimumVersion)
	errInvalidTimeout         = fmt.Errorf("timeout_seconds must be an integer from %d to %d",
		minimumTimeoutSeconds, maximumTimeoutSeconds)
)
