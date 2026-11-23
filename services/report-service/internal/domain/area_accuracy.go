package domain

type AreaAccuracy struct {
	AreaMarginM2       float64
	AreaMarginHectares float64
	Percent            float64
	Confidence         float64
	ScaleShare         float64
	ObstructionShare   float64
	WorstSide          *int
	GroundShiftM       float64
	Basis              string
}

func Centroid(points []Coordinate) Coordinate {
	var centre Coordinate
	for _, point := range points {
		centre.Latitude += point.Latitude / float64(len(points))
		centre.Longitude += point.Longitude / float64(len(points))
	}

	return centre
}
