package scp

import (
	"fmt"
	"testing"
)

func TestAcceptsNominated(t *testing.T) {
	cases := []struct {
		m             Msg
		v             valtype
		wantA, wantVA bool
	}{
		{
			m:      &NomMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &NomMsg{X: ValueSet{valtype(1)}},
			v:      1,
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &NomMsg{Y: ValueSet{valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{B: Ballot{1, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{B: Ballot{1, valtype(1)}},
			v:      2,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{P: Ballot{1, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{P: Ballot{1, valtype(1)}},
			v:      2,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{PP: Ballot{1, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{PP: Ballot{1, valtype(1)}},
			v:      2,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{B: Ballot{1, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &CommitMsg{B: Ballot{1, valtype(1)}},
			v:      2,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtMsg{C: Ballot{1, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &ExtMsg{C: Ballot{1, valtype(1)}},
			v:      2,
			wantA:  false,
			wantVA: false,
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			e := &Env{M: tc.m}
			got := e.acceptsNominated(tc.v)
			if got != tc.wantA {
				t.Errorf("got acceptsNominated=%v, want %v", got, tc.wantA)
			}
			got = e.votesOrAcceptsNominated(tc.v)
			if got != tc.wantVA {
				t.Errorf("got votesOrAcceptsNominated=%v, want %v", got, tc.wantVA)
			}
		})
	}
}

func TestAcceptsPrepared(t *testing.T) {
	cases := []struct {
		m             Msg
		v             valtype
		wantA, wantVA bool
	}{
		{
			m:      &NomMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &NomMsg{X: ValueSet{valtype(1)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &NomMsg{Y: ValueSet{valtype(1)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(1)}},
			v:      1,
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(2)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{P: Ballot{5, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{PP: Ballot{5, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{P: Ballot{5, valtype(2)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{PP: Ballot{5, valtype(2)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(1)}, CN: 6, HN: 10},
			v:      1,
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(1)}, CN: 1, HN: 4},
			v:      1,
			wantA:  false,
			wantVA: true,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(1)}, CN: 1, HN: 10},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(2)}, CN: 6, HN: 10},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(2)}, CN: 1, HN: 4},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &PrepMsg{B: Ballot{5, valtype(2)}, CN: 1, HN: 10},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{B: Ballot{20, valtype(1)}, CN: 10, PN: 10},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{B: Ballot{20, valtype(1)}, CN: 1, PN: 10},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &CommitMsg{B: Ballot{20, valtype(2)}, CN: 1, PN: 10},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{B: Ballot{20, valtype(1)}, CN: 10, PN: 1},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &CommitMsg{B: Ballot{20, valtype(1)}, CN: 10, PN: 5},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &CommitMsg{B: Ballot{20, valtype(2)}, CN: 10, PN: 5},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtMsg{},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtMsg{C: Ballot{1, valtype(1)}},
			v:      1,
			wantA:  true,
			wantVA: true,
		},
		{
			m:      &ExtMsg{C: Ballot{1, valtype(2)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
		{
			m:      &ExtMsg{C: Ballot{10, valtype(1)}},
			v:      1,
			wantA:  false,
			wantVA: false,
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			e := &Env{M: tc.m}
			b := Ballot{5, tc.v}
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
		m        Msg
		v        valtype
		min, max int
		a, va    want
	}{
		{
			m:   &NomMsg{},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &NomMsg{X: ValueSet{valtype(1)}},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &NomMsg{Y: ValueSet{valtype(1)}},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(1)}},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{P: Ballot{1, valtype(1)}},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{PP: Ballot{1, valtype(1)}},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(1)}, CN: 3, HN: 7},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 3, max: 7},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(1)}, CN: 3, HN: 7},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 5, max: 7},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(1)}, CN: 7, HN: 20},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 7, max: 10},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(2)}, CN: 3, HN: 7},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(2)}, CN: 3, HN: 7},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &PrepMsg{B: Ballot{1, valtype(2)}, CN: 7, HN: 20},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &CommitMsg{B: Ballot{1, valtype(1)}, CN: 15, HN: 20},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &CommitMsg{B: Ballot{1, valtype(1)}, CN: 1, HN: 4},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: true, min: 5, max: 10},
		},
		{
			m:   &CommitMsg{B: Ballot{1, valtype(1)}, CN: 1, HN: 7},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: true, min: 5, max: 7},
			va:  want{ok: true, min: 5, max: 7},
		},
		{
			m:   &CommitMsg{B: Ballot{1, valtype(1)}, CN: 4, HN: 12},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: true, min: 5, max: 10},
			va:  want{ok: true, min: 5, max: 10},
		},
		{
			m:   &CommitMsg{B: Ballot{1, valtype(1)}, CN: 7, HN: 12},
			v:   1,
			min: 5,
			max: 10,
			a:   want{ok: true, min: 7, max: 10},
			va:  want{ok: true, min: 7, max: 10},
		},
		{
			m:   &ExtMsg{},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &ExtMsg{C: Ballot{5, valtype(1)}},
			v:   1,
			min: 1,
			max: 10,
			a:   want{ok: true, min: 5, max: 10},
			va:  want{ok: true, min: 5, max: 10},
		},
		{
			m:   &ExtMsg{C: Ballot{5, valtype(1)}},
			v:   1,
			min: 1,
			max: 4,
			a:   want{ok: false},
			va:  want{ok: false},
		},
		{
			m:   &ExtMsg{C: Ballot{5, valtype(1)}},
			v:   1,
			min: 6,
			max: 10,
			a:   want{ok: true, min: 6, max: 10},
			va:  want{ok: true, min: 6, max: 10},
		},
		{
			m:   &ExtMsg{C: Ballot{5, valtype(1)}},
			v:   1,
			min: 3,
			max: 7,
			a:   want{ok: true, min: 5, max: 7},
			va:  want{ok: true, min: 5, max: 7},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			e := &Env{M: tc.m}
			gotOK, gotMin, gotMax := e.acceptsCommit(tc.v, tc.min, tc.max)
			if gotOK != tc.a.ok {
				t.Errorf("got acceptsCommit=%v, want %v", gotOK, tc.a.ok)
			} else if gotOK && (gotMin != tc.a.min || gotMax != tc.a.max) {
				t.Errorf("got min %d, max %d, want min %d, max %d", gotMin, gotMax, tc.a.min, tc.a.max)
			}

			gotOK, gotMin, gotMax = e.votesOrAcceptsCommit(tc.v, tc.min, tc.max)
			if gotOK != tc.va.ok {
				t.Errorf("got votesOrAcceptsCommit=%v, want %v", gotOK, tc.va.ok)
			} else if gotOK && (gotMin != tc.va.min || gotMax != tc.va.max) {
				t.Errorf("got min %d, max %d, want min %d, max %d", gotMin, gotMax, tc.va.min, tc.va.max)
			}
		})
	}
}
