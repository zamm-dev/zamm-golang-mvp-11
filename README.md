# ZAMM MVP: Spec-to-Commit Linking

A Go-based CLI tool for linking specification nodes to Git commits, enabling traceability between requirements and implementation.

## Installation

### Prerequisites

- Go 1.21 or higher

## Quick Start

Build the project:

```bash
make
```

Start ZAMM in interactive mode (with auto-init):

```bash
zamm interactive
```

## Development

Install pre-commit hooks with

```bash
make precommit-hooks
```

Lint with

```bash
make lint
```

Format with

```bash
make fmt
```

Test with

```bash
make test
```
