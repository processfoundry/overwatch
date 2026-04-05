# Contributing to Overwatch

Thanks for your interest. This document is for anyone opening issues or pull requests against this repository.

## What belongs here

Overwatch is: the CLI, TUI, self-hosted server, check execution, and alerting. Changes should fit that scope—small, explicit code paths; respect for operators running the binary without a browser for self-hosted use; clear errors and sensible defaults.

If you are unsure whether an idea fits, open an issue first and describe the use case.

## Development setup

- Install **Go** at the version in [`go.mod`](./go.mod) (or newer, if the project has moved forward).
- Clone the repo and build from the module root:

  ```bash
  go build -o overwatch ./cmd/overwatch
  ```

## Before you open a PR

1. **Format:** `go fmt ./...`
2. **Static check:** `go vet ./...`
3. **Tests:** `go test ./... -race -count=1`

CI runs the same checks on pull requests.

## Pull requests

- Keep changes **focused** on one concern when possible; large refactors are easier to review when split or discussed in an issue first.
- Add or update **tests** when behavior changes or when you fix a bug.
- Match **existing style** in the package you touch (naming, error handling, structure).
- Put **stable, shared types** intended for config or external contracts in [`pkg/spec`](./pkg/spec) rather than deep inside `internal/` unless there is a good reason not to.

## Issues

When reporting bugs, include the **Overwatch version** (`overwatch version`), **OS**, and enough steps or config (redacted) to reproduce. Feature requests are welcome; a short note on the problem you are solving helps prioritize.

## License

By contributing, you agree that your contributions will be licensed under the same terms as the project ([MIT License](./LICENSE)).
