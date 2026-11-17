package landapi

import "time"

type Coordinate struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type Plot struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Points     []Coordinate `json:"points"`
	AreaM2     float64      `json:"area_m2"`
	PerimeterM float64      `json:"perimeter_m"`
	Version    int64        `json:"version"`
	Status     string       `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type PlotDraft struct {
	Name   string       `json:"name"`
	Points []Coordinate `json:"points"`
}

type PlotChanges struct {
	Name   *string       `json:"name,omitempty"`
	Points *[]Coordinate `json:"points,omitempty"`
}

type Report struct {
	ID           string    `json:"id"`
	PlotID       string    `json:"plot_id"`
	PlotName     string    `json:"plot_name"`
	PlotVersion  int64     `json:"plot_version"`
	AreaM2       float64   `json:"area_m2"`
	AreaAres     float64   `json:"area_ares"`
	AreaHectares float64   `json:"area_hectares"`
	PerimeterM   float64   `json:"perimeter_m"`
	CreatedAt    time.Time `json:"created_at"`
}

type createReportRequest struct {
	PlotID string `json:"plot_id"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
