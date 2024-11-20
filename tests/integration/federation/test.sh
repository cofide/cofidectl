#!/bin/bash

# This script deploys two federated trust zones, runs some basic tests against them, then tears them down.
# The trust zones have one attestation policy matching workloads in namespace ns1, and another matching pods with a label foo=bar.

set -euxo pipefail

K8S_CLUSTER_1_NAME=${K8S_CLUSTER_1_NAME:-local1}
K8S_CLUSTER_1_CONTEXT=${K8S_CLUSTER_1_CONTEXT:-kind-$K8S_CLUSTER_1_NAME}

K8S_CLUSTER_2_NAME=${K8S_CLUSTER_2_NAME:-local2}
K8S_CLUSTER_2_CONTEXT=${K8S_CLUSTER_2_CONTEXT:-kind-$K8S_CLUSTER_2_NAME}

TRUST_ZONE_1=${TRUST_ZONE_1:-tz1}
TRUST_DOMAIN_1=${TRUST_DOMAIN_1:-td1}

TRUST_ZONE_2=${TRUST_ZONE_2:-tz2}
TRUST_DOMAIN_2=${TRUST_DOMAIN_2:-td2}

NAMESPACE_POLICY_NAMESPACE=${NAMESPACE_POLICY_NAMESPACE:-demo}
POD_POLICY_POD_LABEL=${POD_POLICY_POD_LABEL:-"foo=bar"}

function configure() {
  rm -f cofide.yaml
  ./cofidectl init
  ./cofidectl trust-zone add $TRUST_ZONE_1 --trust-domain $TRUST_DOMAIN_1 --kubernetes-context $K8S_CLUSTER_1_CONTEXT --kubernetes-cluster $K8S_CLUSTER_1_NAME --profile kubernetes
  ./cofidectl trust-zone add $TRUST_ZONE_2 --trust-domain $TRUST_DOMAIN_2 --kubernetes-context $K8S_CLUSTER_2_CONTEXT --kubernetes-cluster $K8S_CLUSTER_2_NAME --profile kubernetes
  ./cofidectl federation add --from $TRUST_ZONE_1 --to $TRUST_ZONE_2
  ./cofidectl federation add --from $TRUST_ZONE_2 --to $TRUST_ZONE_1
  ./cofidectl attestation-policy add kubernetes --name namespace --namespace $NAMESPACE_POLICY_NAMESPACE
  ./cofidectl attestation-policy add kubernetes --name pod-label --pod-label $POD_POLICY_POD_LABEL
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE_1 --attestation-policy namespace --federates-with $TRUST_ZONE_2
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE_1 --attestation-policy pod-label --federates-with $TRUST_ZONE_2
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE_2 --attestation-policy namespace --federates-with $TRUST_ZONE_1
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE_2 --attestation-policy pod-label --federates-with $TRUST_ZONE_1
}

function up() {
  ./cofidectl up
}

function list_resources() {
  ./cofidectl trust-zone list
  ./cofidectl attestation-policy list
  ./cofidectl attestation-policy-binding list
}

function show_config() {
  cat cofide.yaml
}

function show_status() {
  ./cofidectl workload discover
  ./cofidectl workload list
  ./cofidectl trust-zone status $TRUST_ZONE_1
  ./cofidectl trust-zone status $TRUST_ZONE_2
  ./cofidectl federation list
}

function run_tests() {
  just -f demos/Justfile prompt_namespace=no deploy-ping-pong $K8S_CLUSTER_1_CONTEXT $K8S_CLUSTER_2_CONTEXT
  if ! wait_for_pong; then
    echo "Timed out waiting for pong from server"
    echo "Client logs:"
    kubectl --context $K8S_CLUSTER_1_CONTEXT logs -n demo deployments/ping-pong-client
    echo "Server logs:"
    kubectl --context $K8S_CLUSTER_2_CONTEXT logs -n demo deployments/ping-pong-server
    exit 1
  fi
}

function wait_for_pong() {
  kubectl --context $K8S_CLUSTER_1_CONTEXT wait -n demo --for=condition=Available --timeout 60s deployments/ping-pong-client
  for i in $(seq 30); do
    if kubectl --context $K8S_CLUSTER_1_CONTEXT logs -n demo deployments/ping-pong-client | grep pong; then
      return
    fi
    sleep 2
  done
}

function down() {
  ./cofidectl down
}

function main() {
  configure
  up
  list_resources
  show_config
  show_status
  run_tests
  #down
  echo "Success!"
}

main
