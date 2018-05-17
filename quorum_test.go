package scp

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestFindQuorum(t *testing.T) {
	cases := []struct {
		name    string
		network string
		msgs    string
		want    map[string]string
	}{
		{
			name:    "simple",
			network: "x(a b) a(b x) b(a x)",
			msgs:    "a b x",
			want:    map[string]string{".": "a b x"},
		},
		{
			name:    "cycle",
			network: "a(b) b(x) x(a)",
			msgs:    "a b x",
			want:    map[string]string{".": "a b x", "a": ""},
		},
		{
			name:    "two clusters",
			network: "x(b c / d e) b(x c) c(x b) d(x e) e(x d)",
			msgs:    "x b c d e",
			want:    map[string]string{"b": "x d e", "d": "x b c"},
		},
		{
			name:    "two cycles",
			network: "a(b) b(x) x(a / d) d(e) e(x)",
			msgs:    "a b c d e",
			want:    map[string]string{"d": "a b x", "a": "x d e"},
		},
		{
			network: "x(a z / b) a(z x) b(a x) z(a x)",
			msgs:    "a b z",
			want:    map[string]string{".": "a x z", "b": "a x z", "z": ""},
		},
		{
			name:    "fig 2 with v1=x",
			network: "x(a b) a(b c) b(c a) c(b a)",
			msgs:    "a b c x",
			want:    map[string]string{".": "a b c x", "c": ""},
		},
		{
			name:    "3 of 4",
			network: "x(b c / b d / c d) b(x c / x d / c d) c(x b / x d / b d) d(x b / x c / b c)",
			msgs:    "x b c d",
			want:    map[string]string{"b": "c d x", "c": "b d x", "d": "b c x"},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			network := toNetwork(tc.network)
			ch := make(chan *Msg)
			node := NewNode("x", network["x"], ch, nil)
			slot := newSlot(1, node)
			for _, vstr := range strings.Fields(tc.msgs) {
				v := NodeID(vstr)
				slot.M[v] = &Msg{
					V: v,
					I: 1,
					Q: network[v],
				}
			}
			for exclude, want := range tc.want {
				got := slot.findQuorum(fpred(func(msg *Msg) bool {
					return !strings.Contains(string(msg.V), exclude)
				}))
				want := toNodeIDSet(want)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("got %v, want %v", got, want)
				}
			}
		})
	}
}

func TestFindBlockingSet(t *testing.T) {
	cases := []struct {
		network string
		msgs    string
		want    string
	}{
		{
			network: "x(a b) a(b x) b(a x)",
			msgs:    "a b x",
			want:    "",
		},
		{
			network: "x(a z) a(z x) z(a x)",
			msgs:    "a b z",
			want:    "z",
		},
		{
			network: "x(a z / b) a(z x) b(a x) z(a x)",
			msgs:    "a b z",
			want:    "",
		},
		{
			network: "x(a z / b z) a(z x) b(a x) z(a x)",
			msgs:    "a b z",
			want:    "z",
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			network := toNetwork(tc.network)
			ch := make(chan *Msg)
			node := NewNode("x", network["x"], ch, nil)
			slot := newSlot(1, node)
			for _, vstr := range strings.Fields(tc.msgs) {
				v := NodeID(vstr)
				slot.M[v] = &Msg{
					V: v,
					I: 1,
					Q: network[v],
				}
			}
			got := slot.findBlockingSet(fpred(func(msg *Msg) bool {
				return strings.Contains(string(msg.V), "z")
			}))
			want := toNodeIDSet(tc.want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

// input: "b(a c / d e) c(a b) d(e) e(d)"
// output: map[NodeID][]NodeIDSet{"b": {{"a", "c"}, {"d", "e"}}, "c": {{"a", "b"}}, "d": {{"e"}}, "e": {{"d"}}}
func toNetwork(s string) map[NodeID][]NodeIDSet {
	splitFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		for len(data) > 0 && unicode.IsSpace(rune(data[0])) {
			data = data[1:]
			advance++
		}
		if len(data) == 0 {
			return 0, nil, nil
		}
		switch data[0] {
		case '(', ')', '/':
			return 1 + advance, data[:1], nil
		}
		token = data
		var advance2 int
		for len(data) > 0 && unicode.IsLetter(rune(data[0])) {
			data = data[1:]
			advance2++
		}
		if advance2 == 0 {
			panic("scan error")
		}
		if len(data) == 0 && !atEOF {
			return 0, nil, nil // need to read more to find the end of the token
		}
		return advance + advance2, token[:advance2], nil
	}
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(splitFunc)

	var (
		nodeID   NodeID
		nodeSet  NodeIDSet
		inParen  bool
		nodeSets []NodeIDSet
	)
	result := make(map[NodeID][]NodeIDSet)

	for scanner.Scan() {
		tok := scanner.Text()
		switch tok {
		case "(":
			if inParen || nodeID == "" {
				panic("parse error")
			}
			inParen = true

		case ")":
			if !inParen || len(nodeSet) == 0 {
				panic("parse error")
			}
			nodeSets = append(nodeSets, nodeSet)

			result[nodeID] = nodeSets

			nodeID = ""
			nodeSet = nil
			inParen = false
			nodeSets = nil

		case "/":
			if !inParen || len(nodeSet) == 0 {
				panic("parse error")
			}
			nodeSets = append(nodeSets, nodeSet)
			nodeSet = nil

		default:
			if inParen {
				nodeSet = nodeSet.Add(NodeID(tok))
			} else {
				if nodeID != "" {
					panic("cannot parse")
				}
				nodeID = NodeID(tok)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return result
}
