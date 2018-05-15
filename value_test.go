package scp

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

type valtype uint32

func (v valtype) IsNil() bool { return false }

func (v valtype) Less(other Value) bool {
	return v < other.(valtype)
}

func (v valtype) Combine(other Value, _ SlotID) Value {
	return valtype(v + other.(valtype))
}

func (v valtype) Bytes() []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(v))
	return buf[:]
}

func (v valtype) String() string {
	return strconv.Itoa(int(v))
}

func TestValueSetAdd(t *testing.T) {
	cases := []struct {
		s    []valtype
		v    valtype
		want []valtype
	}{
		{
			s:    []valtype{},
			v:    1,
			want: []valtype{1},
		},
		{
			s:    []valtype{1},
			v:    1,
			want: []valtype{1},
		},
		{
			s:    []valtype{1, 2, 3},
			v:    0,
			want: []valtype{0, 1, 2, 3},
		},
		{
			s:    []valtype{1, 2, 3},
			v:    4,
			want: []valtype{1, 2, 3, 4},
		},
		{
			s:    []valtype{1, 3},
			v:    2,
			want: []valtype{1, 2, 3},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			var vs ValueSet
			for _, val := range tc.s {
				vs = append(vs, val)
			}
			got := vs.Add(tc.v)
			var wantvs ValueSet
			for _, val := range tc.want {
				wantvs = append(wantvs, val)
			}
			if !reflect.DeepEqual(got, wantvs) {
				t.Errorf("got %v, want %v", got, wantvs)
			}
		})
	}
}

func TestValueSetRemove(t *testing.T) {
	cases := []struct {
		s    []valtype
		v    valtype
		want []valtype
	}{
		{
			s:    []valtype{},
			v:    1,
			want: []valtype{},
		},
		{
			s:    []valtype{1},
			v:    1,
			want: []valtype{},
		},
		{
			s:    []valtype{1, 2, 3},
			v:    2,
			want: []valtype{1, 3},
		},
		{
			s:    []valtype{1, 2, 3},
			v:    1,
			want: []valtype{2, 3},
		},
		{
			s:    []valtype{1, 2, 3},
			v:    3,
			want: []valtype{1, 2},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			var vs ValueSet
			for _, val := range tc.s {
				vs = append(vs, val)
			}
			got := vs.Remove(tc.v)
			var wantvs ValueSet
			for _, val := range tc.want {
				wantvs = append(wantvs, val)
			}
			if !reflect.DeepEqual(got, wantvs) {
				t.Errorf("got %v, want %v", got, wantvs)
			}
		})
	}
}

func TestValueSetContains(t *testing.T) {
	cases := []struct {
		s    []valtype
		v    valtype
		want bool
	}{
		{
			s:    []valtype{},
			v:    1,
			want: false,
		},
		{
			s:    []valtype{1},
			v:    1,
			want: true,
		},
		{
			s:    []valtype{1, 2, 3},
			v:    0,
			want: false,
		},
		{
			s:    []valtype{1, 2, 3},
			v:    4,
			want: false,
		},
		{
			s:    []valtype{1, 2, 3},
			v:    2,
			want: true,
		},
		{
			s:    []valtype{1, 3},
			v:    2,
			want: false,
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			var vs ValueSet
			for _, val := range tc.s {
				vs = append(vs, val)
			}
			got := vs.Contains(tc.v)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
