# Contributing to cofidectl

Linting, unit tests and integration tests are run by GitHub Actions against changes proposed to `cofidectl`.
These checks may also be executed locally.

## Running linters

To run the `golangci-lint` checks:

```sh
just lint
```

## Running unit tests

To run Go unit tests:

```sh
just test
```

Or, to run with the race detector enabled (slow):

```sh
just test-race
```

## Running integration tests

There are two integration tests.

### Single trust zone

This test uses `cofidectl` to deploy SPIRE in a single trust zone.
It deploys a ping-pong demo workload and checks that it functions correctly.

Create a Kind cluster.

```sh
just create-kind-cluster
```

Run the `single-trust-zone` integration test.

```sh
just integration-test single-trust-zone
```

### Federated

This test uses `cofidectl` to deploy SPIRE in two federated trust zones.
It deploys a ping-pong demo workload with a server in one trust zone and a client in the other, then checks that it functions correctly.

Create two Kind clusters.

```sh
just create-kind-clusters 2
```

Run the `federation` integration test.

```sh
just integration-test federation
```
