#!/bin/bash

# This script deploys a single trust zone, runs some basic tests against it, then tears it down.
# The trust zone has one attestation policy matching workloads in namespace ns1, and another matching pods with a label foo=bar.

set -euxo pipefail

K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME:-local1}
K8S_CLUSTER_CONTEXT=${K8S_CLUSTER_CONTEXT:-kind-$K8S_CLUSTER_NAME}

TRUST_ZONE=${TRUST_ZONE:-tz1}
TRUST_DOMAIN=${TRUST_DOMAIN:-td1}

NAMESPACE_POLICY_NAMESPACE=${NAMESPACE_POLICY_NAMESPACE:-demo}
POD_POLICY_POD_LABEL=${POD_POLICY_POD_LABEL:-"foo=bar"}

function configure() {
  rm -f cofide.yaml
  ./cofidectl init
  ./cofidectl trust-zone add $TRUST_ZONE --trust-domain $TRUST_DOMAIN --kubernetes-context $K8S_CLUSTER_CONTEXT --kubernetes-cluster $K8S_CLUSTER_NAME --profile kubernetes
  ./cofidectl attestation-policy add kubernetes --name namespace --namespace $NAMESPACE_POLICY_NAMESPACE
  ./cofidectl attestation-policy add kubernetes --name pod-label --pod-label $POD_POLICY_POD_LABEL
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy namespace
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy pod-label
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
  ./cofidectl trust-zone status $TRUST_ZONE
}

function run_tests() {
  just -f demos/Justfile prompt_namespace=no deploy-ping-pong $K8S_CLUSTER_CONTEXT
  if ! wait_for_pong; then
    echo "Timed out waiting for pong from server"
    echo "Client logs:"
    kubectl --context $K8S_CLUSTER_CONTEXT logs -n demo deployments/ping-pong-client
    echo "Server logs:"
    kubectl --context $K8S_CLUSTER_CONTEXT logs -n demo deployments/ping-pong-server
    exit 1
  fi
}

function wait_for_pong() {
  kubectl --context $K8S_CLUSTER_CONTEXT wait -n demo --for=condition=Available --timeout 60s deployments/ping-pong-client
  for i in $(seq 30); do
    if kubectl --context $K8S_CLUSTER_CONTEXT logs -n demo deployments/ping-pong-client | grep pong; then
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
  down
  echo "Success!"
}

main