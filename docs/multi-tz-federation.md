# Introduction

In the [Quickstart](../README.md#quickstart), we saw how to deploy a single trust zone instance using `cofidectl` to secure workloads that are in the same Kubernetes cluster.

In this example, the workloads will instead be in separate trust zones, with distinct roots of trust, and federation with be enabled using `cofidectl`.

## Deploy multi-trust zone Cofide instances

Let's add an additional trust-zone (`cofide-b`) in a new `kind` cluster (in this case, `kind2`) and a Cofide federation between them:

```sh
./cofidectl trust-zone add cofide-b --trust-domain cofide-b.test --kubernetes-cluster kind2 --profile kubernetes --kubernetes-context kind-kind2
./cofidectl federation add --from cofide-a --to cofide-b
./cofidectl federation add --from cofide-b --to cofide-a
```

We'll reuse the existing `namespace-demo` attestation policy for the `cofide-b` trust zone. We also need to rebind it to `cofide-a`, this time enabling federation with `cofide-b`.

```sh
./cofidectl attestation-policy-binding del --attestation-policy namespace-demo --trust-zone-name cofide-a
./cofidectl attestation-policy-binding add --attestation-policy namespace-demo --trust-zone-name cofide-a --federates-with cofide-b
./cofidectl attestation-policy-binding add --attestation-policy namespace-demo --trust-zone-name cofide-b --federates-with cofide-a
```

As before, we apply the configuration using the `up` command:

```sh
./cofidectl up
```

`cofidectl` will take care of the federation itself and initial exchange of trust roots. We can now deploy ping-pong, this time using a different `Justfile` recipe: this example will deploy the ping-pong server to `kind-kind` and the client to `kind-kind2` (in that order).

```sh
just -f demos/Justfile deploy-ping-pong kind-kind kind-kind2
```

Soon you will see the client and server in a game of mutually trusted ping-pong:

```sh
kubectl --context kind-kind logs -n demo deployments/ping-pong-client --follow
```

```sh
2024/11/02 23:35:18 INFO ping...
2024/11/02 23:35:18 INFO ...pong
2024/11/02 23:35:23 INFO ping...
2024/11/02 23:35:23 INFO ...pong
```

The trust zones have been successfully federated and the client and server workloads are securely communicating with mTLS 🔐.
