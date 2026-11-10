package postgres

import (
	"reflect"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestPointsStorageFormat(t *testing.T) {
	points := []domain.Coordinate{{Latitude: 50.45, Longitude: 30.52}, {Latitude: 50.46, Longitude: 30.53}}

	encoded, err := encodePoints(points)
	if err != nil {
		t.Fatalf("encodePoints() error = %v", err)
	}

	if want := `[{"lat":50.45,"lon":30.52},{"lat":50.46,"lon":30.53}]`; string(encoded) != want {
		t.Fatalf("encoded = %s, want %s", encoded, want)
	}

	decoded, err := decodePoints(encoded)
	if err != nil {
		t.Fatalf("decodePoints() error = %v", err)
	}

	if !reflect.DeepEqual(decoded, points) {
		t.Fatalf("decoded = %#v, want %#v", decoded, points)
	}
}
