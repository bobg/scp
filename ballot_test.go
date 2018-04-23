package scp

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBallotSetAdd(t *testing.T) {
	cases := []struct {
		s    []Ballot
		b    Ballot
		want []Ballot
	}{
		{
			s:    []Ballot{},
			b:    Ballot{1, valtype(1)},
			want: []Ballot{{1, valtype(1)}},
		},
		{
			s:    []Ballot{{1, valtype(1)}},
			b:    Ballot{1, valtype(1)},
			want: []Ballot{{1, valtype(1)}},
		},
		{
			s:    []Ballot{{1, valtype(1)}, {1, valtype(2)}, {1, valtype(3)}},
			b:    Ballot{1, valtype(0)},
			want: []Ballot{{1, valtype(0)}, {1, valtype(1)}, {1, valtype(2)}, {1, valtype(3)}},
		},
		{
			s:    []Ballot{{1, valtype(1)}, {1, valtype(2)}, {1, valtype(3)}},
			b:    Ballot{1, valtype(4)},
			want: []Ballot{{1, valtype(1)}, {1, valtype(2)}, {1, valtype(3)}, {1, valtype(4)}},
		},
		{
			s:    []Ballot{{1, valtype(1)}, {1, valtype(3)}},
			b:    Ballot{1, valtype(2)},
			want: []Ballot{{1, valtype(1)}, {1, valtype(2)}, {1, valtype(3)}},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			got := BallotSet(tc.s).Add(tc.b)
			if !reflect.DeepEqual(got, BallotSet(tc.want)) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
