//go:generate go run genset.go
package scp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/davecgh/go-xdr/xdr"
)

type NodeID string

func (n NodeID) Less(other NodeID) bool { return n < other }

// Node is the type of a participating SCP node.
type Node struct {
	ID NodeID

	// Q is the node's set of quorum slices.
	// For compactness,
	// this representation does not include the node itself,
	// though the node is understood to be in every slice.
	Q QSet

	// FP/FQ is a rational giving the odds that a call to Handle will fail (drop the incoming message).
	// Self messages (from the node to itself) are never dropped.
	// FQ==0 is treated as 0/1.
	FP, FQ int

	// mu sync.Mutex

	// pending holds Slot objects during nomination and balloting.
	pending map[SlotID]*Slot

	// ext holds externalized values for slots that have completed
	// balloting.
	ext map[SlotID]*ExtTopic

	cmds *cmdChan
	send chan<- *Msg
}

// NewNode produces a new node.
func NewNode(id NodeID, q QSet, ch chan<- *Msg, ext map[SlotID]*ExtTopic) *Node {
	if ext == nil {
		ext = make(map[SlotID]*ExtTopic)
	}
	return &Node{
		ID:      id,
		Q:       q,
		pending: make(map[SlotID]*Slot),
		ext:     ext,
		cmds:    newCmdChan(),
		send:    ch,
	}
}

// Run processes incoming events for the node. It returns only when
// its context is canceled and should be launched as a goroutine.
func (n *Node) Run(ctx context.Context) {
	var delayUntil *time.Time
	for {
		cmd, ok := n.cmds.read(ctx)
		if !ok {
			if ctx.Err() != nil {
				n.Logf("context canceled, Run exiting")
			} else {
				n.Logf("unknown command-reading error, Run exiting")
			}
			return
		}
		switch cmd := cmd.(type) {
		case *msgCmd:
			func() {
				if delayUntil != nil {
					dur := time.Until(*delayUntil)
					if dur > 0 {
						time.Sleep(dur)
					}
					delayUntil = nil
				}
				err := n.handle(cmd.msg)
				if err != nil {
					n.Logf("ERROR %s", err)
				}
			}()

		case *delayCmd:
			delayUntil = new(time.Time)
			*delayUntil = time.Now().Add(time.Duration(cmd.ms * int(time.Millisecond)))

		case *deferredUpdateCmd:
			func() {
				cmd.slot.deferredUpdate()
			}()

		case *newRoundCmd:
			func() {
				err := cmd.slot.newRound()
				if err != nil {
					n.Logf("ERROR %s", err)
				}
			}()

		case *rehandleCmd:
			func() {
				for _, msg := range cmd.slot.M {
					err := n.handle(msg)
					if err != nil {
						n.Logf("ERROR %s", err)
					}
				}
			}()
		}
	}
}

func (n *Node) deferredUpdate(s *Slot) {
	n.cmds.write(&deferredUpdateCmd{slot: s})
}

func (n *Node) newRound(s *Slot) {
	n.cmds.write(&newRoundCmd{slot: s})
}

func (n *Node) rehandle(s *Slot) {
	n.cmds.write(&rehandleCmd{slot: s})
}

// Handle queues an incoming protocol message. When processed it will
// send a protocol message in response on n.send unless the incoming
// message is ignored. (A message is ignored if it's invalid,
// redundant, or older than another message already received from the
// same sender.)
func (n *Node) Handle(msg *Msg) {
	if msg.V != n.ID && n.FQ > 0 && n.FP < n.FQ {
		// decide whether to simulate dropping this message
		if rand.Intn(n.FQ) < n.FP {
			n.Logf("dropping message %s", msg)
			return
		}
	}
	n.cmds.write(&msgCmd{msg: msg})
}

// Delay simulates a network delay.
func (n *Node) Delay(ms int) {
	n.cmds.write(&delayCmd{ms: ms})
}

func (n *Node) handle(msg *Msg) error {
	if topic, ok := n.ext[msg.I]; ok {
		// This node has already externalized a value for the given slot.
		// Send an EXTERNALIZE message outbound, unless the inbound
		// message is also EXTERNALIZE.
		if inTopic, ok := msg.T.(*ExtTopic); ok {
			// Double check that the inbound EXTERNALIZE value agrees with
			// this node.
			if !ValueEqual(inTopic.C.X, topic.C.X) {
				n.Logf("inbound message %s disagrees with externalized value %s!", msg, topic.C.X)
				panic("consensus failure")
			}
		} else {
			n.send <- NewMsg(n.ID, msg.I, n.Q, topic)
		}
		return nil
	}

	s, ok := n.pending[msg.I]
	if !ok {
		var err error
		s, err = newSlot(msg.I, n)
		if err != nil {
			panic(fmt.Sprintf("cannot create slot %d: %s", msg.I, err))
		}
		n.pending[msg.I] = s
	}

	outbound, err := s.handle(msg)
	if err != nil {
		// delete(n.Pending, msg.I) // xxx ?
		return err
	}

	if outbound == nil {
		return nil
	}

	if extTopic, ok := outbound.T.(*ExtTopic); ok {
		// Handling the inbound message resulted in externalizing a value.
		// We can now save the EXTERNALIZE message and get rid of the Slot
		// object.
		n.ext[msg.I] = extTopic
		delete(n.pending, msg.I)
	}

	n.send <- outbound
	return nil
}

func (n *Node) ping() error {
	for _, s := range n.pending {
		for _, msg := range s.M {
			err := n.handle(msg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ErrNoPrev occurs when trying to compute a hash (with Node.G) for
// slot i before the node has externalized a value for slot i-1.
var ErrNoPrev = errors.New("no previous value")

// G produces a node- and slot-specific 32-byte hash for a given
// message m. It is an error to call this on slot i>1 before n has
// externalized a value for slot i-1.
func (n *Node) G(i SlotID, m []byte) (result [32]byte, err error) { // xxx unexport
	hasher := sha256.New()

	var prevValBytes []byte
	if i > 1 {
		topic, ok := n.ext[i-1]
		if !ok {
			return result, ErrNoPrev
		}
		prevValBytes = topic.C.X.Bytes()
	}

	r, _ := xdr.Marshal(i)
	hasher.Write(r)
	hasher.Write(prevValBytes)
	hasher.Write(m)
	hasher.Sum(result[:0])

	return result, nil
}

// Weight returns the fraction of n's quorum slices in which id
// appears. Return value is the fraction and (as an optimization) a
// bool indicating whether it's exactly 1.
func (n *Node) Weight(id NodeID) (float64, bool) {
	if id == n.ID {
		return 1.0, true
	}
	return n.Q.weight(id)
}

// Peers returns a flattened, uniquified list of the node IDs in n's
// quorum slices, not including n's own ID.
func (n *Node) Peers() NodeIDSet {
	return n.Q.Nodes()
}

// Neighbors produces a deterministic subset of a node's peers (which
// may include itself) that is specific to a given slot and
// nomination-round.
func (n *Node) Neighbors(i SlotID, num int) (NodeIDSet, error) {
	peers := n.Peers()
	peers = peers.Add(n.ID)
	var result NodeIDSet
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
			result = result.Add(nodeID)
		}
	}
	return result, nil
}

// Priority computes a priority for a given peer node that is specific
// to a given slot and nomination-round. The result is a big-endian
// 256-bit integer expressed as a [32]byte.
func (n *Node) Priority(i SlotID, num int, nodeID NodeID) ([32]byte, error) {
	m := new(bytes.Buffer)
	m.WriteByte('P')
	numBytes, _ := xdr.Marshal(num)
	m.Write(numBytes)
	m.WriteString(string(nodeID))
	return n.G(i, m.Bytes())
}

// AllKnown gives the complete set of reachable node IDs,
// excluding n.ID.
func (n *Node) AllKnown() NodeIDSet {
	result := n.Peers()
	for _, s := range n.pending {
		for _, msg := range s.M {
			result = result.Union(msg.Q.Nodes())
		}
	}
	result = result.Remove(n.ID)
	return result
}

// HighestExt returns the ID of the highest slot for which this node
// has an externalized value.
func (n *Node) HighestExt() SlotID {
	var result SlotID
	for slotID := range n.ext {
		if slotID > result {
			result = slotID
		}
	}
	return result
}

// MsgsSince returns all this node's messages with slotID > since.
// TODO: need a better interface, this list could get hella big.
func (n *Node) MsgsSince(since SlotID) []*Msg {
	var result []*Msg

	for slotID, topic := range n.ext {
		if slotID <= since {
			continue
		}
		msg := &Msg{
			V: n.ID,
			I: slotID,
			Q: n.Q,
			T: topic,
		}
		result = append(result, msg)
	}
	for slotID, slot := range n.pending {
		if slotID <= since {
			continue
		}
		result = append(result, slot.Msg())
	}
	return result
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
