---
id: 35d02aa5-0fd2-455e-a955-a589f0af5f0a
slug: directory-structure
type: specification
---

# .zamm directory structure

Project metadata should be contained inside a .zamm directory in the project, with the following structure:

It contains:
- a `nodes/` folder with a separate Markdown file for each node
- a `spec-links.csv` file with links between specs
- a `commit-links.csv` file with links between specs and commits
- a `node-files.csv` file with links between nodes and their files
- a `project_metadata.json` file with project metadata.

---

## Child Specifications

- Children
  - [`node-files.csv`](node-files-csv.md)
