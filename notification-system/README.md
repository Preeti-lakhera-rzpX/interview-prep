# interview-prep

Blank-state Go workspace prepared for the AI-assisted coding assessment.

Contains only generic scaffolding — no problem-specific code.

## Layout
- `go.mod` — module definition.
- `CLAUDE.md` — generic engineering conventions for the agent.
- `Makefile` — build / test / lint targets.
- `.gitignore` — tooling config.

## Common commands
| Command       | What it does                          |
|---------------|---------------------------------------|
| `make build`  | Compile all packages                  |
| `make test`   | Run all tests                         |
| `make race`   | Run tests with the race detector      |
| `make cover`  | Run tests with a coverage report      |
| `make lint`   | gofmt check + `go vet`                |
| `make check`  | lint + test (full gate)               |

## Toolchain
- Go 1.26.1
- Built-in `go test` runner (no external deps)
