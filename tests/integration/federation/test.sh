#!/bin/bash

# This script deploys two federated trust zones, runs some basic tests against them, then tears them down.
# The trust zones have one attestation policy matching workloads in namespace ns1, and another matching pods with a label foo=bar.

set -euxo pipefail

DATA_SOURCE_PLUGIN=${DATA_SOURCE_PLUGIN:-}
PROVISION_PLUGIN=${PROVISION_PLUGIN:-}

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

function check_init() {
  data_source_plugin="$(yq '.plugins.data_source' cofide.yaml -r)"
  provision_plugin="$(yq '.plugins.provision' cofide.yaml -r)"
  if [[ "$data_source_plugin" != "${DATA_SOURCE_PLUGIN:-local}" ]]; then
    echo "Unexpected data source plugin in cofide.yaml: $data_source_plugin vs ${DATA_SOURCE_PLUGIN:-local}"
    exit 1
  fi
  if [[ "$provision_plugin" != "${PROVISION_PLUGIN:-spire-helm}" ]]; then
    echo "Unexpected provision plugin in cofide.yaml: $provision_plugin vs ${PROVISION_PLUGIN:-spire-helm}"
    exit 1
  fi
}

function configure() {
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

  POD_NAME=$(kubectl get pods -l app=ping-pong-client \
    -n $NAMESPACE_POLICY_NAMESPACE \
    -o jsonpath='{.items[0].metadata.name}' \
    --context $K8S_CLUSTER_1_CONTEXT)
  ./cofidectl workload status --namespace $NAMESPACE_POLICY_NAMESPACE \
    --pod-name $POD_NAME \
    --trust-zone $TRUST_ZONE_1
}

function run_tests() {
  just -f demos/Justfile prompt_namespace=no deploy-ping-pong $K8S_CLUSTER_1_CONTEXT $K8S_CLUSTER_2_CONTEXT
  kubectl --context $K8S_CLUSTER_1_CONTEXT wait -n demo --for=condition=Available --timeout 60s deployments/ping-pong-client
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
  for i in $(seq 30); do
    if kubectl --context $K8S_CLUSTER_1_CONTEXT logs -n demo deployments/ping-pong-client | grep '\.\.\.pong'; then
      return 0
    fi
    sleep 2
  done
  return 1
}

function post_deploy() {
  federations=$(./cofidectl federation list)
  if echo "$federations" | grep Unhealthy >/dev/null; then 
      return 1
  fi
}

function down() {
  ./cofidectl down
}

function main() {
  init
  check_init
  configure
  up
  list_resources
  show_config
  show_status
  run_tests
  post_deploy
  down
  echo "Success!"
}

main
