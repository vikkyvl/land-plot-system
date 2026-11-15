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

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createReportRequest struct {
	PlotID string `json:"plot_id"`
}

type loginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
