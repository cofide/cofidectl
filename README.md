# cofidectl: a CLI for Kubernetes workload identity

[![CI](https://github.com/cofide/cofidectl/workflows/ci/badge.svg)](https://github.com/cofide/cofidectl/actions?query=workflow%3Aci+branch%3Amain)

`cofidectl` is a command-line tool that makes it easy to install and manage workload identity providers for Kubernetes, and provide seamless and secure mTLS for applications. It builds on [SPIFFE](https://spiffe.io/docs/latest/spiffe-about/overview/)/[SPIRE](https://spiffe.io/docs/latest/spire-about/) and provides a set of abstractions that make it easy to configure. `cofidectl` can be used to deploy single cluster instances, or handle federation across multiple clusters.

*Note: `cofidectl` is an early-stage project under active development, so please be aware that it is subject to breaking changes.*

## Prerequisites

Pre-built `cofidectl` binaries may be downloaded from the project's [GitHub releases](https://github.com/cofide/cofidectl/releases) page.
Alternatively, `cofidectl` may be built from source code.
Building a `cofidectl` binary requires:

* [Go 1.24 toolchain](https://golang.org/doc/install)
* [`just`](https://github.com/casey/just) as a command runner

To exercise the quickstart requires:

* [`Docker`](https://docs.docker.com/engine/install/)
* [`kind`](https://kind.sigs.k8s.io/docs/user/quick-start)
* [`kubectl`](https://kubernetes.io/docs/tasks/tools/)
* [Cloud provider Kind](https://github.com/kubernetes-sigs/cloud-provider-kind) to expose SPIRE federation endpoints
  * `docker run -d --name cloud-provider-kind --network kind --restart unless-stopped --privileged -v /var/run/docker.sock:/var/run/docker.sock registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.6.0`

## Build

To run the unit tests and build the `cofidectl` binary:

```sh
just build
```

## Quickstart

### Deploy a single trust zone Cofide instance

Deploying to a Kubernetes cluster is as simple as a few commands. This example assumes you have a kind cluster named `kind` and wish to issue SPIFFE identities to workloads for the trust domain `cofide-a.test`'.

```sh
rm -f cofide.yaml
./cofidectl init
./cofidectl trust-zone add cofide-a --trust-domain cofide-a.test --kubernetes-cluster kind --profile kubernetes --kubernetes-context kind-kind
```

Next up is to add an 'attestation policy' - these are `cofidectl` rules which are used to describe the properties of a workload and it's environment to determine workload identity issuance. In this example, we will create a policy (`namespace-demo`) that will enable SPIFFE identities for workloads in the `demo` namespace.

```sh
./cofidectl attestation-policy add kubernetes --name namespace-demo --namespace demo
./cofidectl attestation-policy-binding add --trust-zone cofide-a --attestation-policy namespace-demo
```

Finally, deploy the changes to the cluster:

```sh
./cofidectl up
```

```sh
✅ Installed: Installation completed for cofide-a on cluster kind
✅ Ready: All SPIRE server pods and services are ready for cofide-a in cluster kind
✅ Configured: Post-installation configuration completed for cofide-a on cluster kind
```

And that's how easy it is to get started! 🚀

*If your deployment is stuck on `Waiting for SPIRE server pod and service...`, it may be that you need to restart `cloud-provider-kind` in order for it to create an external IP for your SPIRE server.*

### Deploy an application secured with mTLS

Now let's deploy an application and see how to seamlessly obtain a SPIFFE identity and use it for mTLS.

We've a simple `ping-pong` application with a client that 'pings' and server that responds with 'pong'. For example purposes, the server and client will both reside in a `demo` namespace. The `Justfile` recipes make it quick and easy to apply both:

```sh
just -f demos/Justfile deploy-ping-pong kind-kind
```

Take a look at the logs of the client pod and see the mTLS-enabled ping-pong 🔐:

```sh
kubectl logs -n demo deployments/ping-pong-client --follow
```

```sh
2024/11/02 15:45:50 INFO ping...
2024/11/02 15:45:50 INFO ...pong
2024/11/02 15:45:55 INFO ping...
2024/11/02 15:45:55 INFO ...pong
```

### Deploy multiple federated trust zones

Follow [this guide](docs/multi-tz-federation.md) to see how to configure and deploy Cofide instances in multiple clusters and establish federated trust between workloads that span trust zones.

## Contributing

Contributions are appreciated! See the [contributor guide](CONTRIBUTING.md).

## Production use cases

<div style="float: left; margin-right: 10px;">
    <a href="https://www.cofide.io">
        <img src="docs/img/cofide-colour-blue.svg" width="40" alt="Cofide">
    </a>
</div>

`cofidectl` is a project developed and maintained by [Cofide](https://www.cofide.io). We're building a workload identity platform that is seamless and secure for multi and hybrid cloud environments. If you have a production use case with need for greater flexibility, control and visibility, with enterprise-level support, please [speak with us](mailto:hello@cofide.io) to find out more about the [Cofide](https://www.cofide.io) early access programme 👀.
