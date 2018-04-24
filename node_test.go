package scp

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestPeers(t *testing.T) {
	cases := []struct {
		slices []string
		want   string
	}{
		{},
		{
			slices: []string{"a"},
			want:   "a",
		},
		{
			slices: []string{"a", "a"},
			want:   "a",
		},
		{
			slices: []string{"a b", "a c"},
			want:   "a b c",
		},
		{
			slices: []string{"a b", "c d"},
			want:   "a b c d",
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			var q []NodeIDSet
			for _, slice := range tc.slices {
				ns := toNodeIDSet(slice)
				q = append(q, ns)
			}
			n := NewNode("x", q)
			got := n.Peers()
			want := toNodeIDSet(tc.want)
			if !reflect.DeepEqual(got, NodeIDSet(want)) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestWeight(t *testing.T) {
	cases := []struct {
		slices  []string
		wantW   float64
		wantIs1 bool
	}{
		{
			slices: []string{"a"},
		},
		{
			slices: []string{"a", "b"},
		},
		{
			slices: []string{"a b", "a z"},
			wantW:  0.5,
		},
		{
			slices:  []string{"a b z", "a z"},
			wantW:   1.0,
			wantIs1: true,
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			var q []NodeIDSet
			for _, slice := range tc.slices {
				ns := toNodeIDSet(slice)
				q = append(q, ns)
			}
			n := NewNode("x", q)
			_, is1 := n.Weight(n.ID)
			if !is1 {
				t.Errorf("got !is1, want is1 for n.Weight(n.ID)")
			}
			got, is1 := n.Weight("z")
			if got != tc.wantW || is1 != tc.wantIs1 {
				t.Errorf("got %f (%v), want %f (%v)", got, is1, tc.wantW, tc.wantIs1)
			}
		})
	}
}

func toNodeIDSet(s string) NodeIDSet {
	var result NodeIDSet
	fields := strings.Fields(s)
	for _, f := range fields {
		result = result.Add(NodeID(f))
	}
	return result
}
