package domain

type SelfIntersectionDetails struct {
	Segments [2][2]int
	Swap     [2]int
}

type DuplicatePointDetails struct {
	Indices [2]int
}

type InvalidCoordinateDetails struct {
	Index int
}
