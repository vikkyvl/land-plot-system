package domain

const (
	SquareMetersPerAre     = 100.0
	SquareMetersPerHectare = 10_000.0
)

func SquareMetersToAres(areaM2 float64) float64 { return areaM2 / SquareMetersPerAre }

func SquareMetersToHectares(areaM2 float64) float64 { return areaM2 / SquareMetersPerHectare }
