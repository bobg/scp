package scp

import (
	"testing"
	"time"
)

func TestRound(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want int
	}{
		{0, 1},
		{1 * NomRoundInterval, 1},
		{2 * NomRoundInterval, 1},
		{3 * NomRoundInterval, 2},
		{4 * NomRoundInterval, 2},
		{5 * NomRoundInterval, 2},
		{6 * NomRoundInterval, 2},
		{7 * NomRoundInterval, 3},
	}
	for _, tc := range cases {
		got := round(tc.d)
		if got != tc.want {
			t.Errorf("got round(%s) = %d, want %d", tc.d, got, tc.want)
		}
	}
}
