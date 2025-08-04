---
inclusion: always
---

# Build and Development Commands

## Primary Build Command

Always use `make` (without arguments) as the default build command. This ensures consistency across development environments and CI/CD pipelines.

```bash
# Correct - use this for building
make

# Incorrect - Avoid custom build commands or direct go build calls
go build ./...
```

## Development Workflow Commands

- `make test` - Run all tests with race detection
- `make test-coverage` - Generate HTML coverage reports
- `make fmt` - Format code using standard Go formatting
- `make lint` - Run golangci-lint for code quality checks
- `make clean` - Remove build artifacts from bin/ directory
- `make install` - Install binary globally to GOPATH/bin

## Code Quality Standards

- Always run `make fmt` and `make lint` before committing code
- Use `make test` to ensure all tests pass with race detection enabled
- The build output goes to `bin/zamm` - never commit binaries to version control