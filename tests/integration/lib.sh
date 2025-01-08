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
