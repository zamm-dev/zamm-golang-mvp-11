---
id: a9239511-cbac-4375-b64e-51eb647dcbe5
slug: ""
type: project
---

# Zen and the Automation of Metaprogramming for the Masses

ZAMM is a literate programming tool that tracks the relationship between human-authored requirements and machine-authored code. LLM-generated code often comes with a decent amount of cruft; as such, this tool aims to preserve the human vision for the project inside of a natural language repository with a high SNR. The system is designed for a collaborative workflow between human developers, who define and structure requirements, and LLM assistants, which generate and refactor code based on those specifications. The primary goals are to allow for the easy reimplementation of software across different stacks, and the easy maintenance of forked software regardless of major upstream changes. The primary strategy of achieving these goals is by maintaining a clear and evolving record of project requirements, automating implementation and reimplementation tasks, and preserving the context of architectural decisions as well as the implications of how these decisions interact with individual features.

---

## Child Specifications

- [CLI](cli/README.md)
- [Test Infrastructure](testing/README.md)
- [Link removal should make use of a linked node retrieval function](../.zamm/nodes/218df91d-aba9-4053-b0df-7d1c3bd608ee.md)
- [Commands module should be split up into submodules by command type](../.zamm/nodes/d231c582-fad2-4bab-8352-eef2f46187f8.md)
- [Architecture](architecture/README.md)
- [Data Models](../.zamm/nodes/002e9c7e-8725-480a-b3d6-bc82ae714cb2.md)
- [Project Setup](../.zamm/nodes/02a38b8f-8e66-4dd8-87a3-7b7870f22578.md)
- [Golang CLI Implementation](../.zamm/nodes/8d36673a-0cc9-4484-aa90-7e9670a67f90.md)
- [MCP Server](../.zamm/nodes/ef5c0709-22c7-486f-8fc3-f72a4c9a547b.md)
