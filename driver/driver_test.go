package driver

import (
	"math"
	"testing"
)

func fp(x float64) *float64 { return &x }

func TestCumulativeScartaValoriImpossibili(t *testing.T) {
	for nome, v := range map[string]*float64{
		"nil": nil, "zero": fp(0), "negativo": fp(-12.5), "NaN": fp(math.NaN()), "+Inf": fp(math.Inf(1)), "-Inf": fp(math.Inf(-1)),
	} {
		if got := Cumulative(v); got != nil {
			t.Errorf("%s: atteso nil, ho %v", nome, *got)
		}
	}
}

func TestCumulativeTieneIValoriReali(t *testing.T) {
	for _, want := range []float64{0.1, 150193.9, 54493} {
		if got := Cumulative(fp(want)); got == nil || *got != want {
			t.Errorf("atteso %v, ho %v", want, got)
		}
	}
}
