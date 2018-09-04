/*

Package scp is an implementation of the Stellar Consensus Protocol.

A Node is a participant in an SCP network. A caller feeds incoming
protocol messages (type Msg) to the node's Handle method. In most
cases, the node will respond with another Msg, which the caller should
then disseminate to other network nodes.

The network votes on abstract Value objects proposed by its
members. By means of the protocol, all participating nodes should
eventually converge on a single value for any given "slot." When a
node reaches a final choice of value for a slot, it is said to
"externalize" the value.

A caller may instantiate Value with any concrete type that can be
totally ordered, and for which a deterministic, commutative Combine
operation can be written (reducing two Values to a single one).

A toy demo can be found in cmd/lunch. It takes the name of a TOML file
as an argument. The TOML file specifies the network participants and
topology. Sample TOML files are in cmd/lunch/toml.

*/
package scp
