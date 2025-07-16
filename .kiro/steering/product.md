# Product Overview

ZAMM (Spec-to-Commit Linking) is a Go-based CLI tool that enables traceability between requirements and implementation by linking specification nodes to Git commits.

## Core Features

- **Specification Management**: Create, edit, and organize hierarchical specifications with unique identifiers
- **Git Integration**: Link specifications to Git commit hashes with bidirectional querying
- **Interactive TUI**: Bubble Tea-based terminal user interface for visual spec management
- **Persistent Storage**: File-based JSON storage with CSV linking for portability
- **Hierarchical Organization**: DAG-based parent-child relationships between specifications

## Key Use Cases

- Track which commits implement specific requirements
- Maintain traceability from specs to code changes
- Organize requirements in hierarchical structures
- Query specifications by commit and vice versa
- Support multiple commits per spec and multiple specs per commit

## Architecture Philosophy

The tool follows a clean architecture pattern with clear separation between CLI, services, storage, and models layers. It prioritizes simplicity and portability over complex database dependencies.