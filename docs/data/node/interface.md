---
id: 717c57a1-db3e-4661-90da-980188520bf2
slug: interface
type: specification
---

# Create Node interface and BaseNode struct

We want a Node interface to allow our Golang code to work in predictable ways with different shapes of information.

We want the BaseNode struct to be an anonymous embedded struct so that JSON serialization/deserialization for shared fields stay the same across the struct implementations of different types of nodes
