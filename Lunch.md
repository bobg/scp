# A round of lunch

This repo includes a demonstration program called `lunch`.
It simulates a group of friends or coworkers
(organized into some configurable network of quorum slices)
coming to consensus on what to order for lunch.

This document contains the
(simplified)
output of one round of lunch consensus.
It describes step by step how the network goes from conflicting nominations to agreeing on an outcome.

This example was generated using the [“3 tiers”](https://github.com/bobg/scp/blob/master/cmd/lunch/toml/3tiers.toml) network configuration,
in which:
- a top tier of nodes — Alice, Bob, Carol, and Dave — each depends on two of its neighbors for a quorum;
- a middle tier — Elsie, Fred, Gwen, and Hank — each depends on any two members of the top tier; and
- a bottom tier — Inez and John — each depends on any two members of the middle tier.

Note that lines ending with `-> ∅`
(meaning the node produces no output in response to a protocol message)
are ones the `lunch` program does not normally display,
even though some of them do update the network’s internal state.
They are included here to make the workings of the consensus algorithm clearer.

```
dave: ∅ -> (dave NOM X=[salads], Y=[])
```

Dave votes to nominate salads.
In an SCP protocol message,
X is the set of nominees voted for.

```
elsie: ∅ -> (elsie NOM X=[burgers], Y=[])
```

Elsie votes to nominate burgers.

```
gwen: ∅ -> ∅
```

Gwen would like to nominate something,
but she’s not in her own “high-priority neighbors” list
(at this particular time for this particular slot),
so she discards her own nomination.

```
bob: ∅ -> (bob NOM X=[indian], Y=[])
alice: ∅ -> (alice NOM X=[burritos], Y=[])
```

Bob and Alice vote to nominate Indian food and burritos, respectively.

```
carol: ∅ -> ∅
```

Carol, like Gwen, is not in her own high-priority-neighbors list,
so she does not have the ability to nominate anything at the moment.

```
carol: (dave NOM X=[salads], Y=[]) -> (carol NOM X=[salads], Y=[])
```

However,
Dave _is_ one of Carol’s high-priority neighbors.
She sees his vote to nominate salads and she echoes it.

```
dave: (elsie NOM X=[burgers], Y=[]) -> ∅
elsie: (dave NOM X=[salads], Y=[]) -> ∅
bob: (dave NOM X=[salads], Y=[]) -> ∅
dave: (bob NOM X=[indian], Y=[]) -> ∅
carol: (elsie NOM X=[burgers], Y=[]) -> ∅
alice: (dave NOM X=[salads], Y=[]) -> ∅
```

These nodes all see nominations from their peers but,
throttled by the priority mechanism,
do not echo them.

```
gwen: (dave NOM X=[salads], Y=[]) -> (gwen NOM X=[salads], Y=[])
```

Gwen does echo Dave’s nomination.

```
elsie: (bob NOM X=[indian], Y=[]) -> ∅
carol: (bob NOM X=[indian], Y=[]) -> ∅
bob: (elsie NOM X=[burgers], Y=[]) -> ∅
alice: (elsie NOM X=[burgers], Y=[]) -> ∅
gwen: (elsie NOM X=[burgers], Y=[]) -> ∅
dave: (alice NOM X=[burritos], Y=[]) -> ∅
elsie: (alice NOM X=[burritos], Y=[]) -> ∅
dave: (carol NOM X=[salads], Y=[]) -> ∅
```

Dave sees Carol’s nomination of salads.
Carol may or may not be one of Dave’s high-priority neighbors at the moment,
but in any case Dave has already nominated salads himself,
so this message from Carol does not cause any change in Dave’s state
(which means Dave sends out no new message in response).

```
carol: (alice NOM X=[burritos], Y=[]) -> ∅
gwen: (bob NOM X=[indian], Y=[]) -> ∅
bob: (alice NOM X=[burritos], Y=[]) -> ∅
elsie: (carol NOM X=[salads], Y=[]) -> ∅
dave: (gwen NOM X=[salads], Y=[]) -> ∅
alice: (bob NOM X=[indian], Y=[]) -> ∅
carol: (gwen NOM X=[salads], Y=[]) -> ∅
bob: (carol NOM X=[salads], Y=[]) -> ∅
elsie: (gwen NOM X=[salads], Y=[]) -> ∅
gwen: (alice NOM X=[burritos], Y=[]) -> ∅
alice: (carol NOM X=[salads], Y=[]) -> ∅
bob: (gwen NOM X=[salads], Y=[]) -> ∅
gwen: (carol NOM X=[salads], Y=[]) -> ∅
alice: (gwen NOM X=[salads], Y=[]) -> ∅
```

More throttled-or-redundant peer nominations.

```
hank: ∅ -> (hank NOM X=[salads], Y=[])
```

Hank votes to nominate salads.

```
alice: (hank NOM X=[salads], Y=[]) -> ∅
hank: (dave NOM X=[salads], Y=[]) -> ∅
hank: (elsie NOM X=[burgers], Y=[]) -> ∅
carol: (hank NOM X=[salads], Y=[]) -> ∅
dave: (hank NOM X=[salads], Y=[]) -> ∅
bob: (hank NOM X=[salads], Y=[]) -> ∅
elsie: (hank NOM X=[salads], Y=[]) -> ∅
gwen: (hank NOM X=[salads], Y=[]) -> ∅
hank: (bob NOM X=[indian], Y=[]) -> ∅
hank: (alice NOM X=[burritos], Y=[]) -> ∅
fred: ∅ -> ∅
hank: (carol NOM X=[salads], Y=[]) -> ∅
hank: (gwen NOM X=[salads], Y=[]) -> ∅
fred: (dave NOM X=[salads], Y=[]) -> (fred NOM X=[salads], Y=[])
gwen: (fred NOM X=[salads], Y=[]) -> ∅
fred: (elsie NOM X=[burgers], Y=[]) -> ∅
elsie: (fred NOM X=[salads], Y=[]) -> ∅
dave: (fred NOM X=[salads], Y=[]) -> ∅
hank: (fred NOM X=[salads], Y=[]) -> ∅
alice: (fred NOM X=[salads], Y=[]) -> ∅
carol: (fred NOM X=[salads], Y=[]) -> ∅
bob: (fred NOM X=[salads], Y=[]) -> ∅
fred: (bob NOM X=[indian], Y=[]) -> ∅
fred: (alice NOM X=[burritos], Y=[]) -> ∅
fred: (carol NOM X=[salads], Y=[]) -> ∅
fred: (gwen NOM X=[salads], Y=[]) -> ∅
fred: (hank NOM X=[salads], Y=[]) -> ∅
elsie: (inez NOM X=[pizza], Y=[]) -> ∅
dave: (inez NOM X=[pizza], Y=[]) -> ∅
hank: (inez NOM X=[pizza], Y=[]) -> ∅
inez: (dave NOM X=[salads], Y=[]) -> ∅
inez: (elsie NOM X=[burgers], Y=[]) -> ∅
bob: (inez NOM X=[pizza], Y=[]) -> ∅
gwen: (inez NOM X=[pizza], Y=[]) -> ∅
carol: (inez NOM X=[pizza], Y=[]) -> ∅
fred: (inez NOM X=[pizza], Y=[]) -> ∅
alice: (inez NOM X=[pizza], Y=[]) -> ∅
inez: (bob NOM X=[indian], Y=[]) -> ∅
inez: (alice NOM X=[burritos], Y=[]) -> ∅
inez: (carol NOM X=[salads], Y=[]) -> ∅
inez: (gwen NOM X=[salads], Y=[]) -> ∅
inez: (hank NOM X=[salads], Y=[]) -> ∅
inez: (fred NOM X=[salads], Y=[]) -> ∅
john: ∅ -> (john NOM X=[sandwiches], Y=[])
hank: (john NOM X=[sandwiches], Y=[]) -> ∅
bob: (john NOM X=[sandwiches], Y=[]) -> ∅
inez: (john NOM X=[sandwiches], Y=[]) -> ∅
dave: (john NOM X=[sandwiches], Y=[]) -> ∅
john: (dave NOM X=[salads], Y=[]) -> ∅
fred: (john NOM X=[sandwiches], Y=[]) -> ∅
carol: (john NOM X=[sandwiches], Y=[]) -> ∅
elsie: (john NOM X=[sandwiches], Y=[]) -> ∅
gwen: (john NOM X=[sandwiches], Y=[]) -> ∅
alice: (john NOM X=[sandwiches], Y=[]) -> ∅
john: (elsie NOM X=[burgers], Y=[]) -> ∅
john: (bob NOM X=[indian], Y=[]) -> ∅
john: (alice NOM X=[burritos], Y=[]) -> ∅
elsie: (alice NOM X=[burritos], Y=[]) -> ∅
```

The nomination process continues,
with some nominations echoed,
others throttled,
and some nodes self-censoring.

```
gwen: ∅ -> (gwen NOM X=[pasta salads], Y=[])
```

Gwen,
who originally wanted to nominate something but self-censored because she wasn’t in her own high-priority-neighbors list,
now is.
This happened because enough time elapsed since Gwen began this nomination round that her high-priority list expanded.
She was previously echoing a nomination for pasta but now adds her own nomination, for salads.

```
bob: (john NOM X=[sandwiches], Y=[]) -> ∅
carol: (elsie NOM X=[burgers], Y=[]) -> ∅
bob: (dave NOM X=[salads], Y=[]) -> ∅
carol: (gwen NOM X=[salads], Y=[]) -> ∅
```

As mentioned,
Gwen is now nominating both pasta and salads,
but an older protocol message of hers,
from when she was still nominating only salads,
is only just now reaching Carol.
(Carol does not respond because she is already nominating salads.)

```
alice: ∅ -> ∅
carol: (fred NOM X=[salads], Y=[]) -> ∅
carol: ∅ -> ∅
gwen: (dave NOM X=[salads], Y=[]) -> ∅
carol: (dave NOM X=[salads], Y=[]) -> ∅
gwen: (elsie NOM X=[burgers], Y=[]) -> ∅
carol: (bob NOM X=[indian], Y=[]) -> (carol NOM X=[indian salads], Y=[])
elsie: (hank NOM X=[salads], Y=[]) -> ∅
gwen: (alice NOM X=[burritos], Y=[]) -> ∅
carol: (alice NOM X=[burritos], Y=[]) -> ∅
elsie: ∅ -> ∅
gwen: (carol NOM X=[salads], Y=[]) -> ∅
alice: (bob NOM X=[indian], Y=[]) -> (alice NOM X=[burritos indian], Y=[])
```

Alice’s high-priority-neighbor list has now expanded to include Bob,
and so she starts echoing his nomination of Indian food.

```
gwen: (bob NOM X=[indian], Y=[]) -> ∅
carol: (hank NOM X=[salads], Y=[]) -> ∅
carol: (inez NOM X=[pizza], Y=[]) -> ∅
carol: (john NOM X=[sandwiches], Y=[]) -> ∅
bob: (alice NOM X=[burritos], Y=[]) -> ∅
bob: (carol NOM X=[salads], Y=[]) -> ∅
bob: (inez NOM X=[pizza], Y=[]) -> ∅
elsie: (dave NOM X=[salads], Y=[]) -> ∅
bob: (fred NOM X=[salads], Y=[]) -> ∅
bob: ∅ -> ∅
bob: (elsie NOM X=[burgers], Y=[]) -> ∅
bob: (gwen NOM X=[salads], Y=[]) -> ∅
bob: (hank NOM X=[salads], Y=[]) -> ∅
elsie: (bob NOM X=[indian], Y=[]) -> ∅
elsie: (inez NOM X=[pizza], Y=[]) -> ∅
elsie: (john NOM X=[sandwiches], Y=[]) -> ∅
alice: (gwen NOM X=[salads], Y=[]) -> ∅
elsie: (carol NOM X=[salads], Y=[]) -> ∅
alice: (hank NOM X=[salads], Y=[]) -> ∅
elsie: (gwen NOM X=[salads], Y=[]) -> ∅
alice: (fred NOM X=[salads], Y=[]) -> ∅
gwen: (hank NOM X=[salads], Y=[]) -> ∅
elsie: (fred NOM X=[salads], Y=[]) -> ∅
alice: (dave NOM X=[salads], Y=[]) -> ∅
gwen: (fred NOM X=[salads], Y=[]) -> ∅
alice: (elsie NOM X=[burgers], Y=[]) -> ∅
dave: (fred NOM X=[salads], Y=[]) -> ∅
gwen: (inez NOM X=[pizza], Y=[]) -> ∅
alice: (carol NOM X=[salads], Y=[]) -> ∅
dave: (inez NOM X=[pizza], Y=[]) -> ∅
gwen: (john NOM X=[sandwiches], Y=[]) -> ∅
alice: (inez NOM X=[pizza], Y=[]) -> ∅
dave: (elsie NOM X=[burgers], Y=[]) -> ∅
alice: (john NOM X=[sandwiches], Y=[]) -> ∅
inez: ∅ -> (inez NOM X=[pizza], Y=[])
dave: (bob NOM X=[indian], Y=[]) -> (dave NOM X=[indian salads], Y=[])
dave: (alice NOM X=[burritos], Y=[]) -> ∅
john: (carol NOM X=[salads], Y=[]) -> ∅
dave: (gwen NOM X=[salads], Y=[]) -> ∅
dave: ∅ -> ∅
dave: (carol NOM X=[salads], Y=[]) -> ∅
dave: (hank NOM X=[salads], Y=[]) -> ∅
dave: (john NOM X=[sandwiches], Y=[]) -> ∅
bob: (gwen NOM X=[pasta salads], Y=[]) -> ∅
john: (gwen NOM X=[salads], Y=[]) -> ∅
dave: (gwen NOM X=[pasta salads], Y=[]) -> ∅
inez: (gwen NOM X=[pasta salads], Y=[]) -> ∅
inez: (carol NOM X=[indian salads], Y=[]) -> ∅
bob: (carol NOM X=[indian salads], Y=[]) -> ∅
elsie: (gwen NOM X=[pasta salads], Y=[]) -> ∅
hank: (gwen NOM X=[pasta salads], Y=[]) -> ∅
carol: (gwen NOM X=[pasta salads], Y=[]) -> ∅
elsie: (carol NOM X=[indian salads], Y=[]) -> ∅
```

Plenty more of the same.

```
dave: (carol NOM X=[indian salads], Y=[]) -> (dave NOM X=[salads], Y=[indian])
```

Something new: Dave has moved Indian food from X,
the set of values he’s voting to nominate,
to Y, the set of values he _accepts_ as nominated.
This happens when either:

1. A _quorum_ votes-or-accepts the same value;
2. A _blocking set_ accepts it.

Dave previously had seen Bob vote to nominate Indian food and echoed that nomination.
Now that Dave sees Carol also voting to nominate Indian food,
condition 1 is satisfied:
Bob,
Carol,
and Dave together form one of Dave’s quorums,
all voting for the same thing.

```
gwen: (carol NOM X=[indian salads], Y=[]) -> ∅
john: (hank NOM X=[salads], Y=[]) -> ∅
inez: (alice NOM X=[burritos indian], Y=[]) -> ∅
john: (fred NOM X=[salads], Y=[]) -> ∅
john: (inez NOM X=[pizza], Y=[]) -> ∅
fred: (gwen NOM X=[pasta salads], Y=[]) -> ∅
alice: (gwen NOM X=[pasta salads], Y=[]) -> ∅
hank: (carol NOM X=[indian salads], Y=[]) -> ∅
elsie: (alice NOM X=[burritos indian], Y=[]) -> ∅
inez: (dave NOM X=[indian salads], Y=[]) -> ∅
```

More redundant-or-throttled nominations.

```
bob: (alice NOM X=[burritos indian], Y=[]) -> (bob NOM X=[], Y=[indian])
carol: (alice NOM X=[burritos indian], Y=[]) -> (carol NOM X=[salads], Y=[indian])
```

More “accepting” of the Indian-food nomination.

```
elsie: (dave NOM X=[indian salads], Y=[]) -> ∅
dave: (alice NOM X=[burritos indian], Y=[]) -> ∅
hank: (alice NOM X=[burritos indian], Y=[]) -> ∅
gwen: (alice NOM X=[burritos indian], Y=[]) -> ∅
inez: (dave NOM X=[salads], Y=[indian]) -> ∅
bob: (dave NOM X=[indian salads], Y=[]) -> ∅
hank: (dave NOM X=[indian salads], Y=[]) -> ∅
john: (gwen NOM X=[pasta salads], Y=[]) -> ∅
fred: (carol NOM X=[indian salads], Y=[]) -> ∅
fred: (alice NOM X=[burritos indian], Y=[]) -> ∅
alice: (carol NOM X=[indian salads], Y=[]) -> (alice NOM X=[burritos], Y=[indian])
alice: (dave NOM X=[indian salads], Y=[]) -> ∅
carol: (dave NOM X=[indian salads], Y=[]) -> ∅
```

Nomination continues.

```
dave: (bob NOM X=[], Y=[indian]) -> ∅
```

Dave sees that Bob now “accepts” Indian food as nominated.
This will be important in a moment.

```
alice: (dave NOM X=[salads], Y=[indian]) -> ∅
bob: (dave NOM X=[salads], Y=[indian]) -> ∅
elsie: (dave NOM X=[salads], Y=[indian]) -> ∅
fred: (dave NOM X=[indian salads], Y=[]) -> (fred NOM X=[salads], Y=[indian])
hank: (dave NOM X=[salads], Y=[indian]) -> ∅
john: (carol NOM X=[indian salads], Y=[]) -> ∅
elsie: (bob NOM X=[], Y=[indian]) -> ∅
gwen: (dave NOM X=[indian salads], Y=[]) -> (gwen NOM X=[pasta salads], Y=[indian])
fred: (dave NOM X=[salads], Y=[indian]) -> ∅
```

Nomination continues.

```
dave: (carol NOM X=[salads], Y=[indian]) -> (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0)
```

Dave,
who already “accepted” the Indian food nomination,
now sees that Carol also accepts it.
Dave earlier saw Bob accept it as well.
Bob-Carol-Dave is one of Dave’s quorums,
and when a quorum accepts something, that’s called _confirmation_.

Dave confirms the nomination of Indian food and so is ready to begin _balloting_.
A ballot is a <counter,value> pair,
and balloting is the process of finding a ballot that all nodes can commit to.
This happens through multiple rounds of voting on statements about ballots,
beginning with ruling out ballots all nodes can agree _not_ to commit to — so-called “aborted” ballots.

Dave votes to _prepare_ the ballot <1,indian>.
This means that all lesser ballots are aborted and Dave promises never to commit to them.
(There are no lesser ballots at this stage,
but anyway that’s the meaning of a “prepare” vote.)

```
john: (alice NOM X=[burritos indian], Y=[]) -> ∅
inez: (bob NOM X=[], Y=[indian]) -> ∅
```

For John and Inez, nomination continues.

```
alice: (bob NOM X=[], Y=[indian]) -> (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0)
```

Alice joines Dave in balloting, also voting to prepare <1,indian>.

```
carol: (dave NOM X=[salads], Y=[indian]) -> ∅
bob: (carol NOM X=[salads], Y=[indian]) -> (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0)
inez: (carol NOM X=[salads], Y=[indian]) -> ∅
hank: (bob NOM X=[], Y=[indian]) -> ∅
inez: (alice NOM X=[burritos], Y=[indian]) -> ∅
dave: (alice NOM X=[burritos], Y=[indian]) -> ∅
hank: (carol NOM X=[salads], Y=[indian]) -> ∅
gwen: (dave NOM X=[salads], Y=[indian]) -> ∅
john: (dave NOM X=[indian salads], Y=[]) -> ∅
elsie: (carol NOM X=[salads], Y=[indian]) -> ∅
gwen: (bob NOM X=[], Y=[indian]) -> ∅
fred: (bob NOM X=[], Y=[indian]) -> ∅
inez: (fred NOM X=[salads], Y=[indian]) -> ∅
dave: (fred NOM X=[salads], Y=[indian]) -> ∅
dave: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
alice: (carol NOM X=[salads], Y=[indian]) -> ∅
gwen: (carol NOM X=[salads], Y=[indian]) -> (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0)
bob: (alice NOM X=[burritos], Y=[indian]) -> ∅
carol: (bob NOM X=[], Y=[indian]) -> (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0)
carol: (alice NOM X=[burritos], Y=[indian]) -> ∅
inez: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
hank: (alice NOM X=[burritos], Y=[indian]) -> ∅
gwen: (alice NOM X=[burritos], Y=[indian]) -> ∅
fred: (carol NOM X=[salads], Y=[indian]) -> (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0)
gwen: (fred NOM X=[salads], Y=[indian]) -> ∅
john: (dave NOM X=[salads], Y=[indian]) -> ∅
elsie: (alice NOM X=[burritos], Y=[indian]) -> ∅
inez: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
alice: (fred NOM X=[salads], Y=[indian]) -> ∅
carol: (fred NOM X=[salads], Y=[indian]) -> ∅
carol: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
fred: (alice NOM X=[burritos], Y=[indian]) -> ∅
carol: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
inez: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
dave: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
```

Dave sees that Alice is voting to prepare <1,indian>.
This will be important in a moment.

```
fred: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
alice: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
bob: (fred NOM X=[salads], Y=[indian]) -> ∅
fred: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
alice: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (fred NOM X=[salads], Y=[indian]) -> ∅
gwen: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
elsie: (fred NOM X=[salads], Y=[indian]) -> ∅
john: (bob NOM X=[], Y=[indian]) -> ∅
hank: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
elsie: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
inez: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
```

Nomination and balloting both continue.
As it happens in this example,
all prepare votes are for the same ballot, <1,indian>.
But it’s possible to have competing prepare votes on differing ballots.

```
dave: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0)
```

Dave sees that Bob is voting to prepare <1,indian>.
He too is voting to prepare <1,indian>,
and he previously saw Alice vote the same way.
Alice-Bob-Dave is one of Dave’s quorums,
and a quorum all voting the same way means Dave can now _accept_ that <1,indian> is prepared.
Dave sets P to the value of the highest accepted-prepared ballot.

```
carol: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0)
```

Carol follows suit upon seeing Alice’s prepare vote.

```
elsie: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
fred: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
bob: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
alice: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0)
john: (carol NOM X=[salads], Y=[indian]) -> ∅
gwen: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
dave: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
elsie: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
inez: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
carol: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
inez: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
bob: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
fred: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0)
inez: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
john: (alice NOM X=[burritos], Y=[indian]) -> ∅
alice: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
carol: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
bob: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0)
john: (fred NOM X=[salads], Y=[indian]) -> ∅
gwen: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0)
hank: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
alice: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
john: (gwen NOM X=[pasta salads], Y=[indian]) -> ∅
dave: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
elsie: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
fred: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
dave: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
gwen: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
carol: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
carol: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
john: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
fred: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
inez: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
alice: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
bob: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
fred: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
fred: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
elsie: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
dave: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
```

Nomination and balloting both continue.

```
carol: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
```

Carol sees that Alice accepts ballot <1,indian> as prepared.
Carol does too,
and earlier she saw Dave accept the same thing.
This makes a quorum all accepting the same ballot,
which means <1,indian> is now _confirmed_ prepared.

Carol sets CN and HN to the counters in the lowest and highest confirmed-prepared ballots.
Carol could also in theory have continued to accept new candidates from the nomination phase before now,
but with a confirmed-prepared ballot she no longer can.

By setting CN and HN,
Carol not only notifies her peers that she confirms <1,indian> is prepared,
but also votes to _commit_ to <1,indian>.

Once a commit vote can be confirmed,
a node considers consensus to be achieved and the value in the ballot can be _externalized_ (acted upon).

```
john: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
hank: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
alice: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
bob: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
gwen: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
elsie: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
dave: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
fred: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
bob: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
elsie: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
inez: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
carol: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
john: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
alice: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
dave: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
gwen: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
elsie: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
fred: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
elsie: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
elsie: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
elsie: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
bob: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
dave: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
alice: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
inez: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
carol: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
john: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
dave: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
carol: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
gwen: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
fred: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
bob: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
carol: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
carol: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
gwen: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
gwen: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
dave: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
alice: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
john: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
inez: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
bob: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
dave: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (inez NOM X=[pizza], Y=[]) -> ∅
hank: (john NOM X=[sandwiches], Y=[]) -> ∅
hank: (elsie NOM X=[burgers], Y=[]) -> ∅
hank: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
fred: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
gwen: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
john: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<> PP=<> CN=0 HN=0) -> ∅
elsie: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
elsie: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
gwen: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
```

More nomination and balloting, more accepting and confirming of prepare votes.

```
carol: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1)
```

Carol has now seen a quorum all confirming preparation of the same ballot. (Alice-Carol-Dave.)
Carol now _accepts_ that ballot is committed.

```
john: (dave NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1)
carol: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
alice: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
alice: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (carol NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
fred: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
dave: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1)
gwen: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
elsie: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (alice NOM/PREP X=[burritos], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
inez: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (fred NOM/PREP X=[salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
hank: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
alice: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1)
carol: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (bob NOM/PREP X=[], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
dave: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
fred: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (fred COMMIT B=<1,indian> PN=1 CN=1 HN=1)
hank: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (bob COMMIT B=<1,indian> PN=1 CN=1 HN=1)
inez: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
fred: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
alice: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
gwen: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (elsie COMMIT B=<1,indian> PN=1 CN=1 HN=1)
carol: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
dave: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (gwen NOM/PREP X=[pasta salads], Y=[indian] B=<1,indian> P=<1,indian> PP=<> CN=0 HN=0) -> ∅
alice: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
fred: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
dave: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (inez PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
gwen: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (gwen COMMIT B=<1,indian> PN=1 CN=1 HN=1)
john: (carol PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
carol: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
hank: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
bob: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
fred: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (inez COMMIT B=<1,indian> PN=1 CN=1 HN=1)
dave: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
alice: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (dave PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
dave: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
hank: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
gwen: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
fred: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
carol: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
alice: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
```

More nodes confirm that <1,indian> is prepared.
Some nodes accept that <1,indian> is committed.
Other nodes are still catching up.

```
hank: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (hank EXT C=<1,indian> HN=1)
```

Hank has now seen Alice, Carol, and Dave all accept that <1,indian> is committed.
He is the first to confirm the ballot is committed and so can externalize the value “indian.”

```
inez: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
dave: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
bob: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
elsie: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
gwen: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
inez: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (fred PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
alice: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
alice: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
dave: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (dave EXT C=<1,indian> HN=1)
alice: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
fred: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
carol: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
alice: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (alice EXT C=<1,indian> HN=1)
gwen: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (alice PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
bob: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
inez: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
gwen: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
elsie: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
gwen: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
carol: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (carol EXT C=<1,indian> HN=1)
gwen: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
bob: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
fred: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
elsie: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (elsie EXT C=<1,indian> HN=1)
fred: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
gwen: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (gwen EXT C=<1,indian> HN=1)
bob: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (bob EXT C=<1,indian> HN=1)
inez: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
john: (hank PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (john PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1)
inez: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
inez: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
inez: (fred COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (inez EXT C=<1,indian> HN=1)
fred: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (fred EXT C=<1,indian> HN=1)
john: (elsie PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> (john COMMIT B=<1,indian> PN=1 CN=1 HN=1)
john: (bob PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (gwen PREP B=<1,indian> P=<1,indian> PP=<> CN=1 HN=1) -> ∅
john: (carol COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
john: (hank COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
john: (dave COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
john: (alice COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> ∅
john: (fred COMMIT B=<1,indian> PN=1 CN=1 HN=1) -> (john EXT C=<1,indian> HN=1)
```

The rest of the network catches up. By the end, everyone has externalized “indian.”
