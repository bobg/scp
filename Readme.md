# SCP - A standalone implementation of the Stellar Consensus Protocol

This is an implementation in Go of SCP,
the Stellar Consensus Protocol.
It allows a leaderless network of participating nodes to reach consensus on proposals.

In an SCP network,
time is divided into consecutive _slots_,
each of which produces consensus on a single proposal.
In the early stages of a slot,
nodes are able to nominate proposals they would like the network to adopt.
As the slot progresses,
some proposals may be discarded,
others may be combined,
and by the end,
all nodes agree on one.
After that,
a new slot begins.

In this implementation,
proposals are represented by the abstract type `Value`.
Concrete instantiations of this type may include any data that can be serialized and totally ordered.
It must also be possible to write a deterministic,
commutative `Combine` operation
(reducing two `Values` to a single one).

A toy demo can be found in
[cmd/lunch](https://github.com/bobg/scp/tree/master/cmd/lunch).
It takes the name of a TOML file as an argument.
The TOML file specifies the network participants and topology.
Sample TOML files are in
[cmd/lunch/toml](https://github.com/bobg/scp/tree/master/cmd/lunch/toml).
