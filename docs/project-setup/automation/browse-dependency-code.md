---
id: 23deff9f-24e1-441c-ac75-36673ca9652a
slug: browse-dependency-code
type: specification
---

# Browse dependency code when stuck

Do not keep assuming the problem is in the project code when you don't even know how the underlying dependency works.

For example, if there's a maximum length being enforced on a textarea, but there's no obvious place in the project where that is being set, stop assuming that it is being set in the project itself, and check the underlying textarea dependency to see if there's code there that sets a default limit.
