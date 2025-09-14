---
id: bf36053a-7916-4ac1-ad47-d49522e306b3
slug: initialize-bubbletea-components
type: specification
---

# Initialize bubbletea components on start

Always initialize all components on start and update them with correct data when needed, rather than lazy initialization. This prevents nil pointer crashes during component lifecycle events like SetSize calls.
