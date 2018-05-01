package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bobg/scp"

	"github.com/chain/txvm/crypto/ed25519"
)

type (
	marshaled struct {
		M json.RawMessage
		S []byte // signature over marshaledPayload
	}

	marshaledPayload struct {
		C int
		V string
		I int
		Q [][]string
		T marshaledTopic
	}

	marshaledTopic struct {
		Type        int // scp.Phase values
		X, Y        []string
		B, C, P, PP marshaledBallot
		PN, HN, CN  int
	}

	marshaledBallot struct {
		N int
		X string
	}
)

func marshal(msg *scp.Msg) ([]byte, error) {
	var q [][]string
	for _, slice := range msg.Q {
		var qslice []string
		for _, id := range slice {
			qslice = append(qslice, id)
		}
		q = append(q, qslice)
	}

	var mt marshaledTopic
	switch topic := msg.T.(type) {
	case *scp.NomTopic:
		// xxx build x and y
		mt.X = x
		mt.Y = y

	case *scp.PrepTopic:
		mt.B = marshaledBallot{N: topic.B.N, X: topic.B.X.(valtype).String()}
		mt.P = marshaledBallot{N: topic.P.N, X: topic.P.X.(valtype).String()}
		mt.PP = marshaledBallot{N: topic.PP.N, X: topic.PP.X.(valtype).String()}
		mt.HN = topic.HN
		mt.CN = topic.CN

	case *scp.CommitTopic:
		mt.B = marshaledBallot{N: topic.B.N, X: topic.B.X.(valtype).String()}
		mt.PN = topic.PN
		mt.HN = topic.HN
		mt.CN = topic.CN

	case *scp.ExtTopic:
		mt.C = marshaledBallot{N: topic.C.N, topic.C.X.(valtype).String()}
		mt.HN = topic.HN
	}
	mp := marshaledPayload{
		C: msg.C,
		V: msg.V,
		I: msg.I,
		Q: q,
		T: mt,
	}
	mpbytes, err := json.Marshal(mp) // xxx json is subject to mutation in transit!
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(prv, mpbytes)
	m := marshaled{
		M: mpbytes,
		S: sig,
	}
	return json.Marshal(m)
}

func unmarshal(b []byte) (*scp.Msg, error) {
	var m marshaled
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(pubkey, m.M, m.S) {
		return nil, errors.New("bad signature")
	}

	var mp marshaledPayload
	err = json.Unmarshal(m.M, &mp)
	if err != nil {
		return nil, err
	}

	var q []scp.NodeIDSet
	for _, slice := range mp.Q {
		var qslice scp.NodeIDSet
		for _, id := range slice {
			qslice = qslice.Add(id)
		}
		q = append(q, qslice)
	}

	var topic scp.Topic
	switch mp.T.Type {
	case scp.PhNom:
		topic = &scp.NomTopic{
			X: x,
			Y: y,
		}

	case scp.PhPrep:
		topic = &scp.PrepTopic{
			B:  b,
			P:  p,
			PP: pp,
			HN: mp.T.HN,
			CN: mp.T.CN,
		}

	case scp.PhCommit:
		topic = &scp.CommitTopic{
			B:  b,
			PN: mp.T.PN,
			HN: mp.T.HN,
			CN: mp.T.CN,
		}

	case scp.PhExt:
		topic = &scp.ExtTopic{
			C:  c,
			HN: mp.T.HN,
		}

	default:
		return nil, fmt.Errorf("unknown topic type %d", mp.T.Type)
	}

	msg := &scp.Msg{
		C: mp.C,
		V: mp.V,
		I: mp.I,
		Q: q,
		T: topic,
	}
	return msg, nil
}
