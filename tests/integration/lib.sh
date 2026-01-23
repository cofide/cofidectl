# This file provides common library functions for use by integration tests.

function init() {
  local data_source_plugin=${1:-}
  local provision_plugin=${2:-}
  rm -f cofide.yaml
  local args=()
  if [[ -n "$data_source_plugin" ]]; then
    args+=("--data-source-plugin" "$data_source_plugin")
  fi
  if [[ -n "$provision_plugin" ]]; then
    args+=("--provision-plugin" "$provision_plugin")
  fi
  ./cofidectl init "${args[@]}"
}

function up() {
  local args=()
  for tz in "$@"; do
    args+=("--trust-zone" "$tz")
  done
  ./cofidectl up --quiet "${args[@]}"
}

function down() {
  local args=()
  for tz in "$@"; do
    args+=("--trust-zone" "$tz")
  done
  ./cofidectl down --quiet "${args[@]}"
}

function list_resources() {
  ./cofidectl trust-zone list
  ./cofidectl cluster list
  ./cofidectl attestation-policy list
  ./cofidectl attestation-policy-binding list
}

function show_config() {
  cat cofide.yaml
}

function check_spire_server() {
  local context=${1:?Spire server k8s context}
  if ! kubectl --context $context -n spire-server get statefulsets spire-server; then
    echo "Server statefulset not found"
    return 1
  fi
}

function check_spire_agents() {
  local context=${1:?Spire agent k8s context}
  if ! kubectl --context $context -n spire-system get daemonsets spire-agent; then
    echo "Agent daemonset not found"
    return 1
  fi
}

function check_spire_csi_driver() {
  local context=${1:?Spire CSI k8s context}
  if ! kubectl --context $context -n spire-system get csidrivers.storage.k8s.io csi.spiffe.io; then
    echo "CSI driver not found"
    return 1
  fi
}

function check_delete() {
  trust_zones="$(yq '.trust_zones' cofide.yaml -r)"
  clusters="$(yq '.clusters' cofide.yaml -r)"
  attestation_policies="$(yq '.attestation_policies' cofide.yaml -r)"
  if [[ "$trust_zones" != "null" ]]; then
    echo "Unexpected trust zones in cofide.yaml: $trust_zones"
    exit 1
  fi
  if [[ "$clusters" != "null" ]]; then
    echo "Unexpected clusters in cofide.yaml: $clusters"
    exit 1
  fi
  if [[ "$attestation_policies" != "null" ]]; then
    echo "Unexpected attestation policies in cofide.yaml: $attestation_policies"
    exit 1
  fi
}

function run_ping_pong_test() {
  local client_context=${1:?Client Kubernetes context}
  local client_spiffe_ids="${2:?Client SPIFFE IDs}"
  local server_context=${3:?Server Kubernetes context}
  just -f demos/Justfile prompt_namespace=no deploy-ping-pong $client_context "$client_spiffe_ids" $server_context
  kubectl --context $client_context wait -n demo --for=condition=Available --timeout 60s deployments/ping-pong-client
  if ! wait_for_pong $client_context; then
    echo "Timed out waiting for pong from server"
    echo "Client logs:"
    kubectl --context $client_context logs -n demo deployments/ping-pong-client
    echo "Server logs:"
    kubectl --context $server_context logs -n demo deployments/ping-pong-server
    exit 1
  fi
}

function wait_for_pong() {
  local context=${1:?Kubernetes context}
  for i in $(seq 30); do
    if kubectl --context $context logs -n demo deployments/ping-pong-client | grep '\.\.\.pong'; then
      return 0
    fi
    sleep 2
  done
  return 1
}
