package domain

type SelfIntersectionDetails struct {
	Segments [2][2]int
	Swap     [2]int
	Point    Coordinate
}

type DuplicatePointDetails struct {
	Indices [2]int
}

type InvalidCoordinateDetails struct {
	Index int
}
