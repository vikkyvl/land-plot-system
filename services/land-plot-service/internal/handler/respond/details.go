package respond

import "github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"

type ProblemResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type selfIntersectionResponse struct {
	Segments [2][2]int     `json:"segments"`
	Swap     [2]int        `json:"swap"`
	Point    pointResponse `json:"point"`
}

type pointResponse struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type duplicatePointResponse struct {
	Indices [2]int `json:"indices"`
}

type indexResponse struct {
	Index int `json:"index"`
}

func NewProblemResponse(problem *domain.Error) ProblemResponse {
	return ProblemResponse{Code: problem.Code, Message: problem.Message, Details: detailsResponse(problem.Details)}
}

func detailsResponse(details any) any {
	switch value := details.(type) {
	case domain.SelfIntersectionDetails:
		return selfIntersectionResponse{
			Segments: value.Segments,
			Swap:     value.Swap,
			Point:    pointResponse{Latitude: value.Point.Latitude, Longitude: value.Point.Longitude},
		}
	case domain.DuplicatePointDetails:
		return duplicatePointResponse{Indices: value.Indices}
	case domain.InvalidCoordinateDetails:
		return indexResponse{Index: value.Index}
	case domain.InvalidObstructionDetails:
		return indexResponse{Index: value.Index}
	default:
		return nil
	}
}
