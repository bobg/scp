package scp

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

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
			node := NewNode("x", network["x"], nil)
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
