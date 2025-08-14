---
id: 151affbc-31b0-419e-9cbb-52daa9ddbeaf
title: Common component should separate viewport from rendering concerns
type: specification
---

The inner component, SpecDetail, should be responsible for all spec detail rendering logic.

The outer common component, SpecDetailView, should only contain viewport-related logic. All spec-detail-oriented logic shoulud be rendered inside of SpecDetail itself.
