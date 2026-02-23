package storage

import (
	"testing"
)

func TestParsePointWKT(t *testing.T) {
	tests := []struct {
		name    string
		wkt     string
		wantLat float64
		wantLon float64
		wantErr bool
	}{
		{
			name:    "valid point",
			wkt:     "POINT(-76.456 -12.123)",
			wantLat: -12.123,
			wantLon: -76.456,
		},
		{
			name:    "valid point with whitespace",
			wkt:     "  POINT(-76.456 -12.123)  ",
			wantLat: -12.123,
			wantLon: -76.456,
		},
		{
			name:    "zero coordinates",
			wkt:     "POINT(0 0)",
			wantLat: 0,
			wantLon: 0,
		},
		{
			name:    "positive coordinates",
			wkt:     "POINT(2.3522 48.8566)",
			wantLat: 48.8566,
			wantLon: 2.3522,
		},
		{
			name:    "empty string",
			wkt:     "",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			wkt:     "LINESTRING(-76 -12)",
			wantErr: true,
		},
		{
			name:    "missing closing paren",
			wkt:     "POINT(-76.456 -12.123",
			wantErr: true,
		},
		{
			name:    "invalid longitude",
			wkt:     "POINT(not_a_float -12.123)",
			wantErr: true,
		},
		{
			name:    "invalid latitude",
			wkt:     "POINT(-76.456 not_a_float)",
			wantErr: true,
		},
		{
			name:    "too many coordinates",
			wkt:     "POINT(-76.456 -12.123 0)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lon, err := parsePointWKT(tt.wkt)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePointWKT(%q) error = %v, wantErr %v", tt.wkt, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if lat != tt.wantLat {
				t.Errorf("lat = %v, want %v", lat, tt.wantLat)
			}
			if lon != tt.wantLon {
				t.Errorf("lon = %v, want %v", lon, tt.wantLon)
			}
		})
	}
}

func TestRowToStop(t *testing.T) {
	tests := []struct {
		name     string
		id       int32
		stopName string
		geom     interface{}
		wantErr  bool
		wantLat  float64
		wantLon  float64
	}{
		{
			name:     "valid stop",
			id:       1,
			stopName: "Centro",
			geom:     "POINT(-76.456 -12.123)",
			wantLat:  -12.123,
			wantLon:  -76.456,
		},
		{
			name:     "non-string geom",
			id:       1,
			stopName: "Bad",
			geom:     42,
			wantErr:  true,
		},
		{
			name:     "nil geom (NULL geometry in DB)",
			id:       2,
			stopName: "Bad",
			geom:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid WKT",
			id:       1,
			stopName: "Bad",
			geom:     "LINESTRING(0 0, 1 1)",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stop, err := rowToStop(tt.id, tt.stopName, tt.geom)
			if (err != nil) != tt.wantErr {
				t.Fatalf("rowToStop error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if stop.ID != tt.id {
				t.Errorf("ID = %d, want %d", stop.ID, tt.id)
			}
			if stop.Name != tt.stopName {
				t.Errorf("Name = %q, want %q", stop.Name, tt.stopName)
			}
			if stop.Lat != tt.wantLat {
				t.Errorf("Lat = %v, want %v", stop.Lat, tt.wantLat)
			}
			if stop.Lon != tt.wantLon {
				t.Errorf("Lon = %v, want %v", stop.Lon, tt.wantLon)
			}
		})
	}
}
