package scp

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"log"
	"math/big"
	"sort"
	"sync"

	"github.com/davecgh/go-xdr/xdr"
)

type NodeID string

// Node is the type of a participating SCP node.
type Node struct {
	ID NodeID

	// Q is the node's set of quorum slices. For compactness it does not
	// include the node itself, though the node is understood to be in
	// every slice.
	Q [][]NodeID

	// Pending holds Slot objects during nomination and balloting.
	Pending map[SlotID]*Slot

	// Ext holds externalized values for slots that have completed
	// balloting.
	Ext map[SlotID]*ExtMsg

	mu sync.Mutex
}

// NewNode produces a new node.
func NewNode(id NodeID, q [][]NodeID) *Node {
	return &Node{
		ID:      id,
		Q:       q,
		Pending: make(map[SlotID]*Slot),
		Ext:     make(map[SlotID]*ExtMsg),
	}
}

// Handle processes an incoming protocol message. Returns an outbound
// protocol message in response, or nil if the incoming message is
// ignored. (A message is ignored if it's invalid, redundant, or older
// than another message already received from the same sender.)
func (n *Node) Handle(env *Env) (*Env, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if msg, ok := n.Ext[env.I]; ok {
		// This node has already externalized a value for the given slot.
		// Send an EXTERNALIZE message outbound, unless the inbound
		// message is also EXTERNALIZE.
		// TODO: ...in which case double-check that the values agree?
		if _, ok = env.M.(*ExtMsg); ok {
			return nil, nil
		}
		return NewEnv(n.ID, env.I, n.Q, msg), nil
	}

	var isNew bool
	s, ok := n.Pending[env.I]
	if !ok {
		s = newSlot(env.I, n)
		isNew = true
		n.Pending[env.I] = s
	}

	outbound, err := s.Handle(env)
	if err != nil {
		delete(n.Pending, env.I) // xxx ?
		return nil, err
	}
	if outbound == nil {
		if isNew {
			// This was an attempt to start a new nomination round, and it
			// failed. Discard the new slot.
			delete(n.Pending, env.I)
		}
		return nil, nil
	}

	if extMsg, ok := outbound.M.(*ExtMsg); ok {
		// Handling the inbound message resulted in externalizing a value.
		// We can now save the EXTERNALIZE message and get rid of the Slot
		// object.
		n.Ext[env.I] = extMsg
		delete(n.Pending, env.I)
	}

	return outbound, nil
}

var ErrNoPrev = errors.New("no previous value")

func (n *Node) G(i SlotID, m []byte) (result [32]byte, err error) {
	hasher := sha256.New()

	var prevValBytes []byte
	if i > 1 {
		msg, ok := n.Ext[i-1]
		if !ok {
			return result, ErrNoPrev
		}
		prevValBytes = msg.C.X.Bytes()
	}

	r, _ := xdr.Marshal(i)
	hasher.Write(r)
	hasher.Write(prevValBytes)
	hasher.Write(m)
	hasher.Sum(result[:0])

	return result, nil
}

// Weight returns the fraction of n's quorum slices in which id
// appears.  Return value is the fraction and (as an optimization) a
// bool indicating whether it's exactly 1.
func (n *Node) Weight(id NodeID) (float64, bool) {
	if id == n.ID {
		return 1.0, true
	}
	count := 0
	for _, slice := range n.Q {
		for _, thisID := range slice {
			if id == thisID {
				count++
				break
			}
		}
	}
	if count == len(n.Q) {
		return 1.0, true
	}
	return float64(count) / float64(len(n.Q)), false
}

// Peers returns a flattened, uniquified list of the node IDs in n's
// quorum slices, not including n's own ID.
func (n *Node) Peers() []NodeID {
	var result []NodeID
	for _, slice := range n.Q {
		for _, id := range slice {
			result = append(result, id)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	var (
		to   int
		last NodeID
	)
	for from := 0; from < len(result); from++ {
		if result[from] == last {
			continue
		}
		last = result[from]
		if from != to {
			result[to] = result[from]
		}
		to++
	}
	return result[:to]
}

// Neighbors produces a deterministic subset of a node's peers (which
// may include itself) that is specific to a given slot and
// nomination-round.
func (n *Node) Neighbors(i SlotID, num int) ([]NodeID, error) {
	peers := n.Peers()
	peers = append(peers, n.ID)
	var result []NodeID
	for _, nodeID := range peers {
		weight64, is1 := n.Weight(nodeID)
		var hwBytes []byte
		if is1 {
			hwBytes = maxUint256[:]
		} else {
			w := big.NewFloat(weight64)
			w.Mul(w, hmax)
			hwInt, _ := w.Int(nil)
			hwBytes = hwInt.Bytes()
		}
		var hw [32]byte
		copy(hw[32-len(hwBytes):], hwBytes) // hw is now a big-endian uint256

		m := new(bytes.Buffer)
		m.WriteByte('N')
		numBytes, _ := xdr.Marshal(num)
		m.Write(numBytes)
		m.WriteString(string(nodeID))
		g, err := n.G(i, m.Bytes())
		if err != nil {
			return nil, err
		}
		if bytes.Compare(g[:], hw[:]) < 0 {
			result = append(result, nodeID)
		}
	}
	return result, nil
}

// Priority computes a priority for a given peer node that is specific
// to a given slot and nomination-round.
func (n *Node) Priority(i SlotID, num int, nodeID NodeID) ([32]byte, error) {
	m := new(bytes.Buffer)
	m.WriteByte('P')
	numBytes, _ := xdr.Marshal(num)
	m.Write(numBytes)
	m.WriteString(string(nodeID))
	return n.G(i, m.Bytes())
}

// Logf produces log output prefixed with the node's identity.
func (n *Node) Logf(f string, a ...interface{}) {
	f = "node %s: " + f
	a = append([]interface{}{n.ID}, a...)
	log.Printf(f, a...)
}

var maxUint256 = [32]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// maxuint256, as a float
var hmax *big.Float

func init() {
	hmaxInt := new(big.Int)
	hmaxInt.SetBytes(maxUint256[:])
	hmax = new(big.Float)
	hmax.SetInt(hmaxInt)
}
