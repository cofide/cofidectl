# cofidectl

[![CI](https://github.com/cofide/cofidectl/workflows/ci/badge.svg)](https://github.com/cofide/cofidectl/actions?workflow=ci)

`cofidectl` is a command line tool for installing & administering workload identity provider infrastructure (e.g. SPIRE) for zero trust architectures atop Kubernetes clusters

## Prerequisites

* Uses [`just`](https://github.com/casey/just) as a command runner

## Build

To run the tests and build the `cofidectl` binary

```
just build
```

## Quickstart

Deploying a SPIRE cluster to a k8s context is as simple as:
```
cofidectl init
cofidectl trust-zone add example --trust-domain alpha.test --kubernetes-cluster alpha --profile kubernetes --kubernetes-context kind-alpha
cofidectl up
```
