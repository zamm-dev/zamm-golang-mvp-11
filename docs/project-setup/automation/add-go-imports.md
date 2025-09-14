---
id: 40dce662-6e62-40fd-8136-7a56ee416f0c
slug: add-go-imports
type: specification
---

# Add Go imports after code that uses it

The Go code formatter automatically removes unused imports. If you as an LLM find that the import you're trying to add keeps getting removed right after you add it, then write the code that uses it first before adding the import in.
