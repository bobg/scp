package scp

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAcceptsNominated(t *testing.T) {
	cases := []struct {
		m             Topic
		wantA, wantVA []int
	}{
		{
			m: &NomTopic{},
		},
		{
			m:      &NomTopic{X: ValueSet{valtype(1)}},
			wantVA: []int{1},
		},
		{
			m:      &NomTopic{Y: ValueSet{valtype(1)}},
			wantA:  []int{1},
			wantVA: []int{1},
		},
		{
			m:      &NomTopic{X: ValueSet{valtype(1)}, Y: ValueSet{valtype(2)}},
			wantA:  []int{2},
			wantVA: []int{1, 2},
		},
		{
			m:      &PrepTopic{B: Ballot{1, valtype(1)}},
			wantA:  []int{1},
			wantVA: []int{1},
		},
		{
			m:      &PrepTopic{B: Ballot{1, valtype(1)}, P: Ballot{1, valtype(2)}},
			wantA:  []int{1, 2},
			wantVA: []int{1, 2},
		},
		{
			m:      &PrepTopic{B: Ballot{1, valtype(1)}, P: Ballot{1, valtype(2)}, PP: Ballot{1, valtype(3)}},
			wantA:  []int{1, 2, 3},
			wantVA: []int{1, 2, 3},
		},
		{
			m:      &CommitTopic{B: Ballot{1, valtype(1)}},
			wantA:  []int{1},
			wantVA: []int{1},
		},
		{
			m:      &ExtTopic{C: Ballot{1, valtype(1)}},
			wantA:  []int{1},
			wantVA: []int{1},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			e := &Msg{T: tc.m}
			got := e.acceptsNominatedSet()
			var want ValueSet
			for _, val := range tc.wantA {
				want = want.Add(valtype(val))
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got acceptsNominatedSet=%v, want %v", got, want)
			}
			want = nil
			for _, val := range tc.wantVA {
				want = want.Add(valtype(val))
			}
			got = e.votesOrAcceptsNominatedSet()
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got votesOrAcceptsNominatedSet=%v, want %v", got, want)
			}
		})
	}
}

func TestAcceptsPrepared(t *testing.T) {
	cases := []struct {
		m             Topic
		wantA, wantVA bool
	}{
		{
			m:      &NomTopic{},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &NomTopic{X: ValueSet{valtype(1)}},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &NomTopic{Y: ValueSet{valtype(1)}},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(1)}},
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(2)}},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepTopic{P: Ballot{5, valtype(1)}},
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepTopic{PP: Ballot{5, valtype(1)}},
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepTopic{P: Ballot{5, valtype(2)}},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepTopic{PP: Ballot{5, valtype(2)}},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(1)}, CN: 6, HN: 10},
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(1)}, CN: 1, HN: 4},
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(1)}, CN: 1, HN: 10},
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(2)}, CN: 6, HN: 10},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(2)}, CN: 1, HN: 4},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepTopic{B: Ballot{5, valtype(2)}, CN: 1, HN: 10},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitTopic{B: Ballot{20, valtype(1)}, CN: 10, PN: 10},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitTopic{B: Ballot{20, valtype(1)}, CN: 1, PN: 10},
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &CommitTopic{B: Ballot{20, valtype(2)}, CN: 1, PN: 10},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitTopic{B: Ballot{20, valtype(1)}, CN: 10, PN: 1},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitTopic{B: Ballot{20, valtype(1)}, CN: 10, PN: 5},
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &CommitTopic{B: Ballot{20, valtype(2)}, CN: 10, PN: 5},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtTopic{C: Ballot{1, valtype(1)}},
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &ExtTopic{C: Ballot{1, valtype(2)}},
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtTopic{C: Ballot{10, valtype(1)}},
			wantA:  false,
			wantVA: false,
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			e := &Msg{T: tc.m}
			b := Ballot{5, valtype(1)}
			got := e.acceptsPrepared(b)
			if got != tc.wantA {
				t.Errorf("got acceptsPrepared=%v, want %v", got, tc.wantA)
			}
			got = e.votesOrAcceptsPrepared(b)
			if got != tc.wantVA {
				t.Errorf("got votesOrAcceptsPrepared=%v, want %v", got, tc.wantVA)
			}
		})
	}
}

func TestAcceptsCommit(t *testing.T) {
	type want struct {
		ok       bool
		min, max int
	}

	cases := []struct {
		m        Topic
		min, max int
		a, va    want
	}{
		{
			m:   &NomTopic{},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &NomTopic{X: ValueSet{valtype(1)}},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &NomTopic{Y: ValueSet{valtype(1)}},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(1)}},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(1)}, CN: 3, HN: 7},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 3, max: 7},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(1)}, CN: 3, HN: 7},
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 5, max: 7},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(1)}, CN: 7, HN: 20},
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 7, max: 10},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(2)}, CN: 3, HN: 7},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(2)}, CN: 3, HN: 7},
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepTopic{B: Ballot{1, valtype(2)}, CN: 7, HN: 20},
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &CommitTopic{B: Ballot{1, valtype(1)}, CN: 15, HN: 20},
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &CommitTopic{B: Ballot{1, valtype(1)}, CN: 1, HN: 4},
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 5, max: 10},
		},
		{
			m:   &CommitTopic{B: Ballot{1, valtype(1)}, CN: 1, HN: 7},
			min: 5,
			max: 10,
			a:   want{ok: true, min: 5, max: 7},
			va:  want{ok: true, min: 5, max: 7},
		},
		{
			m:   &CommitTopic{B: Ballot{1, valtype(1)}, CN: 4, HN: 12},
			min: 5,
			max: 10,
			a:   want{ok: true, min: 5, max: 10},
			va:  want{ok: true, min: 5, max: 10},
		},
		{
			m:   &CommitTopic{B: Ballot{1, valtype(1)}, CN: 7, HN: 12},
			min: 5,
			max: 10,
			a:   want{ok: true, min: 7, max: 10},
			va:  want{ok: true, min: 7, max: 10},
		},
		{
			m:   &ExtTopic{C: Ballot{5, valtype(1)}},
			min: 1,
			max: 10,
			a:   want{ok: true, min: 5, max: 10},
			va:  want{ok: true, min: 5, max: 10},
		},
		{
			m:   &ExtTopic{C: Ballot{5, valtype(1)}},
			min: 1,
			max: 4,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &ExtTopic{C: Ballot{5, valtype(1)}},
			min: 6,
			max: 10,
			a:   want{ok: true, min: 6, max: 10},
			va:  want{ok: true, min: 6, max: 10},
		},
		{
			m:   &ExtTopic{C: Ballot{5, valtype(1)}},
			min: 3,
			max: 7,
			a:   want{ok: true, min: 5, max: 7},
			va:  want{ok: true, min: 5, max: 7},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			e := &Msg{T: tc.m}
			gotOK, gotMin, gotMax := e.acceptsCommit(valtype(1), tc.min, tc.max)
			if gotOK != tc.a.ok {
				t.Errorf("got acceptsCommit=%v, want %v", gotOK, tc.a.ok)
			} else if gotOK && (gotMin != tc.a.min || gotMax != tc.a.max) {
				t.Errorf("got min %d, max %d, want min %d, max %d", gotMin, gotMax, tc.a.min, tc.a.max)
			}

			gotOK, gotMin, gotMax = e.votesOrAcceptsCommit(valtype(1), tc.min, tc.max)
			if gotOK != tc.va.ok {
				t.Errorf("got votesOrAcceptsCommit=%v, want %v", gotOK, tc.va.ok)
			} else if gotOK && (gotMin != tc.va.min || gotMax != tc.va.max) {
				t.Errorf("got min %d, max %d, want min %d, max %d", gotMin, gotMax, tc.va.min, tc.va.max)
			}
		})
	}
}
