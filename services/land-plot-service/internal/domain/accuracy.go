package domain

import "math"

const (
	groundResolutionAtEquatorM = 156543.03
	AccuracyConfidence         = 0.95
	accuracyConfidenceFactor   = 2
	areaVarianceDivisor        = 4
	zoomScaleBase              = 2
	percentMultiplier          = 100
)

type AccuracyModel struct {
	MouseClickPixels float64
	TouchClickPixels float64
	MapShiftM        float64
	SatelliteShiftM  float64
	TreesM           float64
	BuildingsM       float64
	UnclearM         float64
	FallbackCapture  Capture
}

type AccuracyContributions struct {
	Scale        float64
	Obstructions float64
}

type Accuracy struct {
	AreaMarginM2       float64
	AreaMarginHectares float64
	Percent            float64
	Confidence         float64
	Contributions      AccuracyContributions
	WorstSide          *int
	GroundShiftM       float64
}

type vertexObstruction struct {
	sigmaM float64
	side   int
}

func (model AccuracyModel) Estimate(
	points []Coordinate,
	areaM2 float64,
	survey Survey,
) Accuracy {
	used := model.FallbackCapture
	if survey.Capture != nil {
		used = *survey.Capture
	}

	clickSigma := model.clickPixels(used.Pointer) * groundResolution(points, used.Zoom)
	projected := projectToLocalPlane(points)
	vertexObstructions := model.vertexObstructions(survey.Obstructions, len(points))

	var scaleVariance, obstructionVariance float64

	sideVariance := make(map[int]float64)

	for i := range projected {
		weight := neighbourSpanSquared(projected, i) / areaVarianceDivisor
		scaleVariance += clickSigma * clickSigma * weight

		obstruction := vertexObstructions[i]
		if obstruction.sigmaM > 0 {
			variance := obstruction.sigmaM * obstruction.sigmaM * weight
			obstructionVariance += variance
			sideVariance[obstruction.side] += variance
		}
	}

	margin := accuracyConfidenceFactor * math.Sqrt(scaleVariance+obstructionVariance)

	return Accuracy{
		AreaMarginM2:       margin,
		AreaMarginHectares: SquareMetersToHectares(margin),
		Percent:            margin / areaM2 * percentMultiplier,
		Confidence:         AccuracyConfidence,
		Contributions:      contributions(scaleVariance, obstructionVariance),
		WorstSide:          worstSide(sideVariance),
		GroundShiftM:       model.groundShift(used.Layer),
	}
}

func (model AccuracyModel) vertexObstructions(obstructions []Obstruction, pointCount int) []vertexObstruction {
	vertices := make([]vertexObstruction, pointCount)

	for _, obstruction := range obstructions {
		sigma := model.obstructionSigma(obstruction.Reason)
		for _, vertex := range []int{obstruction.Side, (obstruction.Side + 1) % pointCount} {
			current := vertices[vertex]
			if strongerObstruction(sigma, obstruction.Side, current) {
				vertices[vertex] = vertexObstruction{sigmaM: sigma, side: obstruction.Side}
			}
		}
	}

	return vertices
}

func strongerObstruction(sigma float64, side int, current vertexObstruction) bool {
	if sigma != current.sigmaM {
		return sigma > current.sigmaM
	}

	return current.sigmaM > 0 && side < current.side
}

func (model AccuracyModel) clickPixels(pointer Pointer) float64 {
	if pointer == PointerTouch {
		return model.TouchClickPixels
	}

	return model.MouseClickPixels
}

func (model AccuracyModel) groundShift(layer MapLayer) float64 {
	if layer == LayerSatellite {
		return model.SatelliteShiftM
	}

	return model.MapShiftM
}

func (model AccuracyModel) obstructionSigma(reason ObstructionReason) float64 {
	switch reason {
	case ReasonTrees:
		return model.TreesM
	case ReasonBuildings:
		return model.BuildingsM
	case ReasonUnclear:
		return model.UnclearM
	default:
		return 0
	}
}

func groundResolution(points []Coordinate, zoom float64) float64 {
	var latitudeSum float64
	for _, point := range points {
		latitudeSum += point.Latitude
	}

	latitude := degreesToRadians(latitudeSum / float64(len(points)))

	return groundResolutionAtEquatorM * math.Cos(latitude) / math.Pow(zoomScaleBase, zoom)
}

func neighbourSpanSquared(projected []metricPoint, index int) float64 {
	count := len(projected)
	next := projected[(index+1)%count]
	previous := projected[(index-1+count)%count]
	deltaX := next.x - previous.x
	deltaY := next.y - previous.y

	return deltaX*deltaX + deltaY*deltaY
}

func contributions(scaleVariance, obstructionVariance float64) AccuracyContributions {
	total := scaleVariance + obstructionVariance
	if total == 0 {
		return AccuracyContributions{}
	}

	return AccuracyContributions{
		Scale:        scaleVariance / total,
		Obstructions: obstructionVariance / total,
	}
}

func worstSide(sideVariance map[int]float64) *int {
	var worst *int

	for side, variance := range sideVariance {
		if worst == nil || variance > sideVariance[*worst] || (variance == sideVariance[*worst] && side < *worst) {
			chosen := side
			worst = &chosen
		}
	}

	return worst
}
