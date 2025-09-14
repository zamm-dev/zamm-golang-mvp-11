---
id: 27c7d9af-29fc-49c9-b241-00cc575b178d
slug: data-structure
type: specification
---

# Child node groupings should be specified as a map of child label to child node

This is because, given the present node as the head of the tree, the children are going to be a list-equivalent instead of a node. The list-equivalent should be a list of label-child tuples, or in other words a mapping from label to a list of nodes matching that label, because all child nodes with the same label will get lumped together.
