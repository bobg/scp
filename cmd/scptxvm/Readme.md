# scptxvm

> “Hey, you got your consensus protocol in my blockchain representation!”
>
> “You got your blockchain representation in my consensus protocol!”

This is scptxvm, an experimental implementation of a decentralized
blockchain using Chain’s TxVM as the transaction and block model, and
the Stellar Consensus Protocol for achieving network consensus.

The program manages a single SCP node. It is configured with one or
more other nodes as its set of “quorum slices.” Each node’s ID is the
URL at which it listens for inbound SCP messages. Nodes vote on which
block should be chosen for each block height.

SCP messages contain block IDs only. Each node may request the actual
contents of a block, given its ID, from any other.  No node may pass
along a block ID without knowing the block’s contents, in case some
other node asks for it. Nodes also use block contents when combining
two valid proposed blocks into a new block proposal.

The program also listens for proposed transactions, which are added to
a transaction pool and periodically rolled up into a proposed block.
The ease with which a node can get its pool transactions included in
an adopted block depends heavily on the topology of the network.
