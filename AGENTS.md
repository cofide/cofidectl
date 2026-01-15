# AGENTS.md: `cofidectl`

## Project Overview

`cofidectl` is a command-line tool written in Go for managing workload identity in Kubernetes. It builds on [SPIFFE](https://spiffe.io/)/[SPIRE](https://spiffe.io/docs/latest/spire-about/) to provide mutual TLS (mTLS) for applications and simplifies the configuration and deployment of single and multi-cluster federated identity providers.

The tool utilizes a plugin architecture, allowing for extensibility. Key technologies used include:

*   **Go**: The primary programming language.
*   **Cobra**: For building the command-line interface.
*   **SPIFFE/SPIRE**: For workload identity.
*   **Kubernetes**: As the target environment.
*   **Helm**: For package management in Kubernetes.
*   **just**: As a command runner for development tasks.

## Building and Running

Key commands for development and testing are defined in the `Justfile`.

### Build

To build the `cofidectl` binary from source:

```sh
just build
```

This will compile the application and create a `cofidectl` executable in the root directory.

### Test

To run the unit tests:

```sh
just test
```

To run the tests with the Go race detector enabled (as done in CI):

```sh
just test-race
```

### Linting

To run the linter to check for code style and errors:

```sh
just lint
```

## Development Conventions

*   **CLI Commands**: New commands are added using the `cobra` library. The main command structure is defined in `cmd/cofidectl/cmd/`.
*   **Linting**: The project uses `golangci-lint` for linting. The configuration is in `.golangci.yaml`. All new code should pass the linting checks.
*   **Continuous Integration**: The CI pipeline, defined in `.github/workflows/ci.yml`, runs linting and tests (including race detection) on every push and pull request.
*   **Plugins**: The application supports a plugin model. Plugins are executables with names prefixed by `cofidectl-` and are located in `~/.cofide/plugins`.
