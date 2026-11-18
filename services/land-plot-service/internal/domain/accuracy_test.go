package domain

import (
	"errors"
	"math"
	"testing"
)

const (
	testLatitude          = 50.45
	testLongitude         = 30.52
	metersPerDegree       = 111_320.0
	testSideM             = 157.48
	testAnalyticSigmaM    = 2.0
	testAnalyticSideM     = 100.0
	testMarginToleranceHa = 0.002
)

var testAccuracyModel = AccuracyModel{
	MouseClickPixels: 3,
	TouchClickPixels: 8,
	MapShiftM:        2,
	SatelliteShiftM:  4,
	TreesM:           3,
	BuildingsM:       2,
	UnclearM:         5,
	FallbackCapture:  Capture{Zoom: 16, Layer: LayerSatellite, Pointer: PointerTouch},
}

func testSquare() []Coordinate {
	sideM := testSideM
	latitudeStep := sideM / metersPerDegree
	longitudeStep := sideM / (metersPerDegree * math.Cos(degreesToRadians(testLatitude)))

	return []Coordinate{
		{Latitude: testLatitude, Longitude: testLongitude},
		{Latitude: testLatitude, Longitude: testLongitude + longitudeStep},
		{Latitude: testLatitude + latitudeStep, Longitude: testLongitude + longitudeStep},
		{Latitude: testLatitude + latitudeStep, Longitude: testLongitude},
	}
}

func estimate(points []Coordinate, capture *Capture, obstructions []Obstruction) Accuracy {
	area := LocalEllipsoidCalculator{}.Calculate(points).AreaM2

	return testAccuracyModel.Estimate(points, area, capture, obstructions)
}

func TestAreaVarianceOfSquareMatchesAnalyticResult(t *testing.T) {
	square := []metricPoint{
		{0, 0}, {testAnalyticSideM, 0}, {testAnalyticSideM, testAnalyticSideM}, {0, testAnalyticSideM},
	}

	var variance float64
	for i := range square {
		variance += testAnalyticSigmaM * testAnalyticSigmaM * neighbourSpanSquared(square, i) / areaVarianceDivisor
	}

	expected := math.Sqrt2 * testAnalyticSigmaM * testAnalyticSideM
	if math.Abs(math.Sqrt(variance)-expected) > 1e-9 {
		t.Fatalf("sigma_A = %f, want %f", math.Sqrt(variance), expected)
	}
}

func TestEstimateMatchesReadmeExample(t *testing.T) {
	cases := []struct {
		pointer  Pointer
		marginHa float64
	}{
		{PointerMouse, 0.051},
		{PointerTouch, 0.135},
	}

	for _, testCase := range cases {
		accuracy := estimate(testSquare(), &Capture{Zoom: 18, Layer: LayerMap, Pointer: testCase.pointer}, nil)

		if math.Abs(accuracy.AreaMarginHectares-testCase.marginHa) > testMarginToleranceHa {
			t.Errorf("%s: margin %f ha, want %f ha", testCase.pointer, accuracy.AreaMarginHectares, testCase.marginHa)
		}

		if accuracy.Contributions.Scale != 1 || accuracy.WorstSide != nil || accuracy.GroundShiftM != 2 {
			t.Errorf("%s: unexpected accuracy %+v", testCase.pointer, accuracy)
		}
	}
}

func TestPenCountsAsMouse(t *testing.T) {
	points := testSquare()
	mouse := estimate(points, &Capture{Zoom: 18, Layer: LayerMap, Pointer: PointerMouse}, nil)
	pen := estimate(points, &Capture{Zoom: 18, Layer: LayerMap, Pointer: PointerPen}, nil)

	if mouse.AreaMarginM2 != pen.AreaMarginM2 {
		t.Fatalf("pen margin %f, mouse margin %f", pen.AreaMarginM2, mouse.AreaMarginM2)
	}
}

func TestHigherZoomReducesMargin(t *testing.T) {
	points := testSquare()
	far := estimate(points, &Capture{Zoom: 16, Layer: LayerMap, Pointer: PointerMouse}, nil)
	near := estimate(points, &Capture{Zoom: 19, Layer: LayerMap, Pointer: PointerMouse}, nil)

	if near.AreaMarginM2 >= far.AreaMarginM2 {
		t.Fatalf("zoom 19 margin %f is not below zoom 16 margin %f", near.AreaMarginM2, far.AreaMarginM2)
	}
}

func TestObstructionIncreasesMarginAndNamesWorstSide(t *testing.T) {
	points := testSquare()
	capture := &Capture{Zoom: 18, Layer: LayerSatellite, Pointer: PointerMouse}
	unmarked := estimate(points, capture, nil)
	marked := estimate(
		points,
		capture,
		[]Obstruction{{Side: 1, Reason: ReasonBuildings}, {Side: 2, Reason: ReasonUnclear}},
	)

	if marked.AreaMarginM2 <= unmarked.AreaMarginM2 {
		t.Fatalf("marked margin %f is not above clear margin %f", marked.AreaMarginM2, unmarked.AreaMarginM2)
	}

	if marked.WorstSide == nil || *marked.WorstSide != 2 {
		t.Fatalf("worst side %v, want 2", marked.WorstSide)
	}

	sum := marked.Contributions.Scale + marked.Contributions.Obstructions
	if math.Abs(sum-1) > 1e-9 || marked.Contributions.Obstructions <= marked.Contributions.Scale {
		t.Fatalf("contributions %+v", marked.Contributions)
	}

	if marked.GroundShiftM != 4 {
		t.Fatalf("ground shift %f, want 4", marked.GroundShiftM)
	}
}

func TestSharedVertexTakesLargerObstruction(t *testing.T) {
	vertices := testAccuracyModel.vertexObstructions(
		[]Obstruction{
			{Side: 0, Reason: ReasonTrees},
			{Side: 1, Reason: ReasonUnclear},
			{Side: 3, Reason: ReasonBuildings},
		},
		4,
	)

	expected := []vertexObstruction{{3, 0}, {5, 1}, {5, 1}, {2, 3}}
	for i, vertex := range vertices {
		if vertex != expected[i] {
			t.Errorf("vertex %d: %+v, want %+v", i, vertex, expected[i])
		}
	}
}

func TestMissingCaptureUsesFallback(t *testing.T) {
	points := testSquare()
	missing := estimate(points, nil, nil)
	fallback := estimate(points, &testAccuracyModel.FallbackCapture, nil)

	if missing != fallback {
		t.Fatalf("missing capture %+v, fallback %+v", missing, fallback)
	}
}

func TestValidateCapture(t *testing.T) {
	valid := Capture{Zoom: 18, Layer: LayerSatellite, Pointer: PointerTouch}
	if err := ValidateCapture(&valid); err != nil {
		t.Fatalf("valid capture: %v", err)
	}

	if err := ValidateCapture(nil); err != nil {
		t.Fatalf("missing capture: %v", err)
	}

	invalid := []Capture{
		{Zoom: MaximumZoom + 1, Layer: LayerMap, Pointer: PointerMouse},
		{Zoom: math.NaN(), Layer: LayerMap, Pointer: PointerMouse},
		{Zoom: 18, Layer: "terrain", Pointer: PointerMouse},
		{Zoom: 18, Layer: LayerMap, Pointer: "stylus"},
	}
	for _, capture := range invalid {
		if err := ValidateCapture(&capture); !errors.Is(err, ErrInvalidCapture) {
			t.Errorf("%+v: error %v, want invalid_capture", capture, err)
		}
	}
}

func TestValidateObstructions(t *testing.T) {
	if err := ValidateObstructions(
		[]Obstruction{{Side: 0, Reason: ReasonTrees}, {Side: 3, Reason: ReasonUnclear}},
		4,
	); err != nil {
		t.Fatalf("valid obstructions: %v", err)
	}

	invalid := [][]Obstruction{
		{{Side: 4, Reason: ReasonTrees}},
		{{Side: -1, Reason: ReasonTrees}},
		{{Side: 0, Reason: "fog"}},
		{{Side: 1, Reason: ReasonTrees}, {Side: 1, Reason: ReasonBuildings}},
	}
	for _, obstructions := range invalid {
		err := ValidateObstructions(obstructions, 4)

		var domainError *Error
		if !errors.As(err, &domainError) || domainError.Code != ErrInvalidObstruction.Code ||
			domainError.Kind != KindInvalidRequest {
			t.Errorf("%+v: error %v, want invalid_obstruction", obstructions, err)
		}
	}
}
