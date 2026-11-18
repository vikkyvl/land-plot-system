package domain

import "math"

const (
	MinimumZoom = 0.0
	MaximumZoom = 22.0
)

type MapLayer string

const (
	LayerMap       MapLayer = "map"
	LayerSatellite MapLayer = "satellite"
)

type Pointer string

const (
	PointerMouse Pointer = "mouse"
	PointerTouch Pointer = "touch"
	PointerPen   Pointer = "pen"
)

type ObstructionReason string

const (
	ReasonTrees     ObstructionReason = "trees"
	ReasonBuildings ObstructionReason = "buildings"
	ReasonUnclear   ObstructionReason = "unclear"
)

type Capture struct {
	Zoom    float64
	Layer   MapLayer
	Pointer Pointer
}

type Obstruction struct {
	Side   int
	Reason ObstructionReason
}

type InvalidObstructionDetails struct {
	Index int `json:"index"`
}

func ValidateCapture(capture *Capture) error {
	if capture == nil {
		return nil
	}

	validZoom := !math.IsNaN(capture.Zoom) && capture.Zoom >= MinimumZoom && capture.Zoom <= MaximumZoom
	validLayer := capture.Layer == LayerMap || capture.Layer == LayerSatellite
	validPointer := capture.Pointer == PointerMouse || capture.Pointer == PointerTouch || capture.Pointer == PointerPen

	if !validZoom || !validLayer || !validPointer {
		return ErrInvalidCapture
	}

	return nil
}

func ValidateObstructions(obstructions []Obstruction, pointCount int) error {
	markedSides := make(map[int]bool, len(obstructions))

	for index, obstruction := range obstructions {
		validSide := obstruction.Side >= 0 && obstruction.Side < pointCount
		if !validSide || !knownReason(obstruction.Reason) || markedSides[obstruction.Side] {
			return ErrInvalidObstruction.WithDetails(InvalidObstructionDetails{Index: index})
		}

		markedSides[obstruction.Side] = true
	}

	return nil
}

func knownReason(reason ObstructionReason) bool {
	return reason == ReasonTrees || reason == ReasonBuildings || reason == ReasonUnclear
}
