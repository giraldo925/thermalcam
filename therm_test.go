package main

import (
	"testing"
)

func TestColorIndex(t *testing.T) {
	var temps []float64
	for f := 20.0; f < 50.0; f = f + 0.25 {
		temps = append(temps, f)
	}
	for _, temp := range temps {
		i := getColorIndex(temp)
		if i > len(colors)-1 || i < 0 {
			t.Error("Color index out of range", i)
		}
	}
}
