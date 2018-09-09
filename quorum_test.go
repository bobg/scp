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
			ch := make(chan *Msg)
			node := NewNode("x", slicesToQSet(network["x"]), ch, nil)
			slot, _ := newSlot(1, node)
			for _, vstr := range strings.Fields(tc.msgs) {
				v := NodeID(vstr)
				slot.M[v] = &Msg{
					V: v,
					I: 1,
					Q: slicesToQSet(network[v]),
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

func TestFindQuorum(t *testing.T) {
	cases := []struct {
		id   NodeID
		q    QSet
		m    map[NodeID]*Msg
		want NodeIDSet
	}{
		{
			id:   "x",
			q:    QSet{T: 0},
			m:    map[NodeID]*Msg{},
			want: NodeIDSet{"x"},
		},
		{
			id: "x",
			q: QSet{
				T: 1,
				M: []QSetMember{
					{N: nodeIDPtr("x1")},
					{N: nodeIDPtr("y1")},
				},
			},
			m: map[NodeID]*Msg{
				"x1": &Msg{
					V: "x1",
					Q: QSet{T: 0},
				},
				"y1": &Msg{
					V: "y1",
					Q: QSet{T: 0},
				},
			},
			want: NodeIDSet{"x", "x1"},
		},
		{
			id: "x",
			q: QSet{
				T: 1,
				M: []QSetMember{
					{N: nodeIDPtr("x1")},
					{N: nodeIDPtr("y1")},
				},
			},
			m: map[NodeID]*Msg{
				"x1": &Msg{
					V: "x1",
					Q: QSet{
						T: 1,
						M: []QSetMember{
							{N: nodeIDPtr("y1")},
							{N: nodeIDPtr("z1")},
						},
					},
				},
				"y1": &Msg{
					V: "y1",
					Q: QSet{T: 0},
				},
			},
			want: nil,
		},
		{
			id: "x",
			q: QSet{
				T: 2,
				M: []QSetMember{
					{N: nodeIDPtr("x1")},
					{N: nodeIDPtr("y1")},
					{N: nodeIDPtr("x2")},
				},
			},
			m: map[NodeID]*Msg{
				"x1": &Msg{
					V: "x1",
					Q: QSet{T: 0},
				},
				"y1": &Msg{
					V: "y1",
					Q: QSet{T: 0},
				},
				"x2": &Msg{
					V: "x2",
					Q: QSet{T: 0},
				},
			},
			want: NodeIDSet{"x", "x1", "x2"},
		},
		{
			id: "x",
			q: QSet{
				T: 2,
				M: []QSetMember{
					{
						Q: &QSet{
							T: 1,
							M: []QSetMember{
								{N: nodeIDPtr("x1")},
								{N: nodeIDPtr("y1")},
							},
						},
					},
					{
						Q: &QSet{
							T: 1,
							M: []QSetMember{
								{N: nodeIDPtr("x2")},
								{N: nodeIDPtr("y2")},
							},
						},
					},
				},
			},
			m: map[NodeID]*Msg{
				"x1": &Msg{
					V: "x1",
					Q: QSet{T: 0},
				},
				"y1": &Msg{
					V: "y1",
					Q: QSet{T: 0},
				},
				"x2": &Msg{
					V: "x2",
					Q: QSet{T: 0},
				},
			},
			want: NodeIDSet{"x", "x1", "x2"},
		},
		{
			id: "x",
			q: QSet{
				T: 2,
				M: []QSetMember{
					{N: nodeIDPtr("x1")},
					{N: nodeIDPtr("y2")},
					{
						Q: &QSet{
							T: 2,
							M: []QSetMember{
								{N: nodeIDPtr("x3")},
								{N: nodeIDPtr("y4")},
								{N: nodeIDPtr("y5")},
							},
						},
					},
					{
						Q: &QSet{
							T: 2,
							M: []QSetMember{
								{N: nodeIDPtr("x6")},
								{N: nodeIDPtr("y7")},
								{N: nodeIDPtr("x8")},
							},
						},
					},
					{N: nodeIDPtr("y9")},
				},
			},
			m: map[NodeID]*Msg{
				"x1": &Msg{V: "x1", Q: QSet{T: 0}},
				"y2": &Msg{V: "y2", Q: QSet{T: 0}},
				"x3": &Msg{V: "x3", Q: QSet{T: 0}},
				"y4": &Msg{V: "y4", Q: QSet{T: 0}},
				"y5": &Msg{V: "y5", Q: QSet{T: 0}},
				"x6": &Msg{V: "x6", Q: QSet{T: 0}},
				"y7": &Msg{V: "y7", Q: QSet{T: 0}},
				"x8": &Msg{V: "x8", Q: QSet{T: 0}},
				"y9": &Msg{V: "y9", Q: QSet{T: 0}},
			},
			want: NodeIDSet{"x", "x1", "x6", "x8"},
		},
		{
			id: "x",
			q: QSet{
				T: 2,
				M: []QSetMember{
					{N: nodeIDPtr("x1")},
					{N: nodeIDPtr("y2")},
					{
						Q: &QSet{
							T: 2,
							M: []QSetMember{
								{N: nodeIDPtr("x3")},
								{N: nodeIDPtr("y4")},
								{N: nodeIDPtr("y5")},
							},
						},
					},
					{
						Q: &QSet{
							T: 2,
							M: []QSetMember{
								{N: nodeIDPtr("x6")},
								{N: nodeIDPtr("y7")},
								{N: nodeIDPtr("x8")},
							},
						},
					},
					{N: nodeIDPtr("y9")},
				},
			},
			m: map[NodeID]*Msg{
				"x1": &Msg{V: "x1", Q: QSet{T: 0}},
				"y2": &Msg{V: "y2", Q: QSet{T: 0}},
				"x3": &Msg{V: "x3", Q: QSet{T: 0}},
				"y4": &Msg{V: "y4", Q: QSet{T: 0}},
				"y5": &Msg{V: "y5", Q: QSet{T: 0}},
				"x6": &Msg{
					V: "x6",
					Q: QSet{
						T: 1,
						M: []QSetMember{
							{N: nodeIDPtr("x3")},
							{N: nodeIDPtr("y4")},
						},
					},
				},
				"y7": &Msg{V: "y7", Q: QSet{T: 0}},
				"x8": &Msg{V: "x8", Q: QSet{T: 0}},
				"y9": &Msg{V: "y9", Q: QSet{T: 0}},
			},
			want: NodeIDSet{"x", "x1", "x3", "x6", "x8"},
		},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			got, _ := tc.q.findQuorum(tc.id, tc.m, fpred(func(msg *Msg) bool {
				return msg.V[0] == tc.id[0]
			}))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func nodeIDPtr(s string) *NodeID {
	return (*NodeID)(&s)
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
