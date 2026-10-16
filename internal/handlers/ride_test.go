package handlers

import (
	"math"
	"testing"
)

func TestValidLatitude(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  bool
	}{
		{"zero latitude", 0, true},
		{"valid positive latitude", 32.7266, true},
		{"valid negative latitude", -45.5, true},
		{"north pole", 90, true},
		{"south pole", -90, true},
		{"above maximum", 90.1, false},
		{"below minimum", -90.1, false},
		{"not a number", math.NaN(), false},
		{"positive infinity", math.Inf(1), false},
		{"negative infinity", math.Inf(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validLatitude(tt.value)
			if got != tt.want {
				t.Errorf(
					"validLatitude(%v) = %v; want %v",
					tt.value,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestValidLongitude(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  bool
	}{
		{"zero longitude", 0, true},
		{"valid positive longitude", 74.857, true},
		{"valid negative longitude", -73.9, true},
		{"eastern boundary", 180, true},
		{"western boundary", -180, true},
		{"above maximum", 180.1, false},
		{"below minimum", -180.1, false},
		{"not a number", math.NaN(), false},
		{"positive infinity", math.Inf(1), false},
		{"negative infinity", math.Inf(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validLongitude(tt.value)
			if got != tt.want {
				t.Errorf(
					"validLongitude(%v) = %v; want %v",
					tt.value,
					got,
					tt.want,
				)
			}
		})
	}
}
