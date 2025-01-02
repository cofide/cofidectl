#!/bin/bash

# This script deploys a single trust zone, runs some basic tests against it, then tears it down.
# The trust zone has one attestation policy matching workloads in namespace ns1, and another matching pods with a label foo=bar.

set -euxo pipefail

DATA_SOURCE_PLUGIN=${DATA_SOURCE_PLUGIN:-}
PROVISION_PLUGIN=${PROVISION_PLUGIN:-}

K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME:-local1}
K8S_CLUSTER_CONTEXT=${K8S_CLUSTER_CONTEXT:-kind-$K8S_CLUSTER_NAME}

TRUST_ZONE=${TRUST_ZONE:-tz1}
TRUST_DOMAIN=${TRUST_DOMAIN:-td1}

NAMESPACE_POLICY_NAMESPACE=${NAMESPACE_POLICY_NAMESPACE:-demo}
POD_POLICY_POD_LABEL=${POD_POLICY_POD_LABEL:-"foo=bar"}

function init() {
  rm -f cofide.yaml
  args=""
  if [[ -n "$DATA_SOURCE_PLUGIN" ]]; then
    args="$args --data-source-plugin $DATA_SOURCE_PLUGIN"
  fi
  if [[ -n "$PROVISION_PLUGIN" ]]; then
    args="$args --provision-plugin $PROVISION_PLUGIN"
  fi
  ./cofidectl init $args
}

function configure() {
  ./cofidectl trust-zone add $TRUST_ZONE --trust-domain $TRUST_DOMAIN --kubernetes-context $K8S_CLUSTER_CONTEXT --kubernetes-cluster $K8S_CLUSTER_NAME --profile kubernetes
  ./cofidectl attestation-policy add kubernetes --name namespace --namespace $NAMESPACE_POLICY_NAMESPACE
  ./cofidectl attestation-policy add kubernetes --name pod-label --pod-label $POD_POLICY_POD_LABEL
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy namespace
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy pod-label
}

function up() {
  ./cofidectl up --quiet
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
  kubectl --context $K8S_CLUSTER_CONTEXT wait -n demo --for=condition=Available --timeout 60s deployments/ping-pong-client
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
  for i in $(seq 30); do
    if kubectl --context $K8S_CLUSTER_CONTEXT logs -n demo deployments/ping-pong-client | grep '\.\.\.pong'; then
      return 0
    fi
    sleep 2
  done
  return 1
}

function show_workload_status() {
  POD_NAME=$(kubectl get pods -l app=ping-pong-client \
    -n $NAMESPACE_POLICY_NAMESPACE \
    -o jsonpath='{.items[0].metadata.name}' \
    --context $K8S_CLUSTER_CONTEXT)
  WORKLOAD_STATUS_RESPONSE=$(./cofidectl workload status --namespace $NAMESPACE_POLICY_NAMESPACE \
    --pod-name $POD_NAME \
    --trust-zone $TRUST_ZONE)

  if [[ ! $WORKLOAD_STATUS_RESPONSE == *"SVID verified against trust bundle"* ]]; then
    echo "cofidectl workload status unsuccessful"
    exit 1
  fi

  echo "cofidectl workload status successful"
  exit 0
}

function down() {
  ./cofidectl down
}

function main() {
  init
  configure
  up
  list_resources
  show_config
  show_status
  run_tests
  show_workload_status
  down
  echo "Success!"
}

main
