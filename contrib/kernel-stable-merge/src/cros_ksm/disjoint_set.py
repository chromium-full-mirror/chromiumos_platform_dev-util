# Copyright 2026 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""\
Disjoint set forest that supports union-join and iteration.

Performance of the following operations, as an aggregate, have amortized
constant runtime with low coefficient in all realistic scenarios: `Node.merge`,
`Node.root`, `Node.contains_node`. The iterator `Node.iter_nodes` has amortized
linear runtime O(N) where N is the count of nodes in the node's disjoint set.

The iteration method is an extension to the usual implementation of a disjoint
set forest. It preserves the expected amortized-linear performance of the base
methods by implementing iteration with a singly linked list that is updated
during merge operations.

See Also:
    - https://en.wikipedia.org/w/index.php?title=Disjoint-set_data_structure&oldid=1302977477
"""

from collections.abc import Iterator
from typing import Self


class Node[T]:
    """A node in a disjoint set forest."""

    data: T

    # For the disjoint set tree.
    _parent: Self
    _rank: int

    # For the circular linked list.
    _next: Self

    def __init__(self, data: T):
        self.data = data
        self._parent = self
        self._rank = 0
        self._next = self

    def contains_node(self, other: Self, /) -> bool:
        """Test if this node's set contain the other node."""
        return self.root() is other.root()

    def iter_nodes(self, *, exclude_self=False) -> Iterator[Self]:
        """Iterate over all nodes in this node's set."""
        # pylint: disable=protected-access

        if not exclude_self:
            yield self

        node = self._next
        while node is not self:
            yield node
            node = node._next

    def root(self) -> Self:
        """Return the root node of this node's set.

        This is defined as a normal method and not a property, to hint that the
        method may do work on each call.
        """
        # pylint: disable=protected-access,line-too-long
        #
        # Implement path compression as single-pass path splitting.
        #
        # References:
        #   - https://en.wikipedia.org/w/index.php?title=Disjoint-set_data_structure&oldid=1302977477
        #   - Tarjan, Robert E.; van Leeuwen, Jan (1984). "Worst-case analysis
        #     of set union algorithms". Journal of the ACM. 31 (2): 245–281.
        #     doi:10.1145/62.2160. S2CID 5363073.
        node = self
        while node._parent is not node:
            node, node._parent = node._parent, node._parent._parent
        return node

    def merge(self, other: Self, /) -> Self:
        """Merge this node's set with the other node's set.

        Return the new root node.
        """

        # pylint: disable=protected-access

        x = self.root()
        y = other.root()

        if x is y:
            return x

        # First, merge the trees. The node with higher rank will become the new
        # root.

        if x._rank >= y._rank:
            high, low = x, y
        else:
            high, low = y, x

        if high._rank == low._rank:
            high._rank += 1

        low._parent = high

        # Next, merge the circular lists.
        high._next, low._next = low._next, high._next

        return high
