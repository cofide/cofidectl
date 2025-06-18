# This file provides common library functions for use by integration tests.

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
