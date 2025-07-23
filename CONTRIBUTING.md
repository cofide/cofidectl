# Contributing to cofidectl

Contributions to `cofidectl` are welcome and should be made by GitHub pull requests to https://github.com/cofide/cofidectl.
Google Gemini Code Assist provides automated code reviews.
Changes must pass checks and be approved by a member of the Cofide team before they can be merged.

Linting, unit tests and integration tests are run by GitHub Actions against changes proposed to `cofidectl`.
These checks may also be executed locally.

## Running linters

[`golangci-lint v1`](https://golangci-lint.run/welcome/install/#binaries) must be installed in order to lint `cofidectl`.

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

Running integration tests requires the following:

* [Helm](https://helm.sh/docs/intro/install/)
* [yq](https://github.com/mikefarah/yq)

As described in Kind's [known issues](https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files), it is possible to hit Linux open file limits in pods when using multiple Kind clusters or clusters with several nodes.
This can be avoided as follows:

```
cat << EOF | sudo tee /etc/sysctl.d/10-kind.conf
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 512
EOF
sudo sysctl -p /etc/sysctl.d/10-kind.conf
```

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
