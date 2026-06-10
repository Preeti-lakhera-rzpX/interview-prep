# Project Conventions

Generic engineering conventions for this Go project. No domain-specific logic lives here.

## Language & Tooling
- Go (`go.mod` module: `interview-prep`).
- Format with `gofmt` (enforced). Vet with `go vet`.
- Build, test, and lint via the `Makefile` targets.

## Code Style
- Small, single-responsibility functions. Prefer composition over large monoliths.
- Exported identifiers get doc comments; keep them short and factual.
- Return errors, don't panic, except for truly unrecoverable programmer errors.
- Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- Keep package APIs minimal; unexported by default.
- No premature abstraction — three similar lines beat a wrong abstraction.

## Concurrency
- Document the concurrency model of any shared type (who locks what).
- Prefer channels or `sync` primitives explicitly; never leave data races.
- Run tests with `-race` during development.

## Testing
- Table-driven tests using the standard `testing` package.
- Every exported behavior has at least one test.
- Test the edge cases and failure paths, not just the happy path.
- Tests must be deterministic — no sleeps for synchronization; use channels/sync.
- Target meaningful coverage of state transitions and error handling.

## Workflow Expectations
- Read every generated diff before accepting it.
- Keep changes modular and reviewable; reject monolithic dumps.
- Commit in small, logically-scoped increments.
