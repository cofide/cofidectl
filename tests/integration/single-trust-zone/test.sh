#!/bin/bash

# This script deploys a single trust zone, runs some basic tests against it, then tears it down.
# The trust zone has one attestation policy matching workloads in namespace ns1, and another matching pods with a label foo=bar.

set -euxo pipefail

source $(dirname $(dirname $BASH_SOURCE))/lib.sh

DATA_SOURCE_PLUGIN=${DATA_SOURCE_PLUGIN:-}
PROVISION_PLUGIN=${PROVISION_PLUGIN:-}

K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME:-local1}
K8S_CLUSTER_CONTEXT=${K8S_CLUSTER_CONTEXT:-kind-$K8S_CLUSTER_NAME}

TRUST_ZONE=${TRUST_ZONE:-tz1}
TRUST_DOMAIN=${TRUST_DOMAIN:-td1}

NAMESPACE_POLICY_NAMESPACE=${NAMESPACE_POLICY_NAMESPACE:-demo}
POD_POLICY_POD_LABEL=${POD_POLICY_POD_LABEL:-"foo=bar"}

function configure() {
  ./cofidectl trust-zone add $TRUST_ZONE --trust-domain $TRUST_DOMAIN --no-cluster
  ./cofidectl cluster add $K8S_CLUSTER_NAME --trust-zone $TRUST_ZONE --kubernetes-context $K8S_CLUSTER_CONTEXT --profile kubernetes
  ./cofidectl attestation-policy add kubernetes --name namespace --namespace $NAMESPACE_POLICY_NAMESPACE --spiffeid-path-template 'ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}'
  ./cofidectl attestation-policy add kubernetes --name pod-label --pod-label $POD_POLICY_POD_LABEL --dnsNameTemplates example.namespace.svc.cluster.local
  ./cofidectl attestation-policy add static --name static-namespace --spiffeid spiffe://$TRUST_DOMAIN/ns/$NAMESPACE_POLICY_NAMESPACE/sa/ping-pong-client --selectors k8s:ns:$NAMESPACE_POLICY_NAMESPACE --yes
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy namespace
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy pod-label
  ./cofidectl attestation-policy-binding add --trust-zone $TRUST_ZONE --attestation-policy static-namespace
  override_helm_values
}

function override_helm_values() {
  cat <<EOF >values.yaml
tornjak-frontend:
  enabled: false
upstream-spire-agent:
  upstream: false
EOF
  ./cofidectl trust-zone helm override $TRUST_ZONE --input-file values.yaml
  rm -f values.yaml
}

function check_spire() {
  check_spire_server $K8S_CLUSTER_CONTEXT
  check_spire_agents $K8S_CLUSTER_CONTEXT
  check_spire_csi_driver $K8S_CLUSTER_CONTEXT
}

function show_status() {
  ./cofidectl workload discover
  ./cofidectl workload list
  ./cofidectl trust-zone status $TRUST_ZONE
}

function run_tests() {
  local client_spiffe_id="spiffe://$TRUST_DOMAIN/ns/demo/sa/ping-pong-client"
  run_ping_pong_test $K8S_CLUSTER_CONTEXT $client_spiffe_id $K8S_CLUSTER_CONTEXT
}

function show_workload_status() {
  POD_NAME=$(kubectl get pods -l app=ping-pong-client \
    -n $NAMESPACE_POLICY_NAMESPACE \
    -o jsonpath='{.items[0].metadata.name}' \
    --context $K8S_CLUSTER_CONTEXT)
  WORKLOAD_STATUS_RESPONSE=$(./cofidectl workload status --namespace $NAMESPACE_POLICY_NAMESPACE \
    --pod-name $POD_NAME \
    --trust-zone $TRUST_ZONE)

  if [[ $WORKLOAD_STATUS_RESPONSE != *"SVID verified against trust bundle"* ]]; then
    echo "cofidectl workload status unsuccessful"
    exit 1
  fi

  echo "cofidectl workload status successful"
}

function check_overridden_values() {
  echo "Generated Helm values:"
  ./cofidectl trust-zone helm values $TRUST_ZONE --output-file -

  check_overridden_value '."tornjak-frontend".enabled' "false"
  check_overridden_value '."upstream-spire-agent".upstream' "false"
}

function check_overridden_value() {
  value=$(helm --kube-context $K8S_CLUSTER_CONTEXT get values spire --namespace spire-mgmt | yq $1)
  if [[ $value != $2 ]]; then
    echo "Error: Did not find expected overridden Helm value $1: expected $2, actual $value"
    return 1
  fi
}

function check_dns_name_templates() {
  expected="- example.namespace.svc.cluster.local" 
  value=$(kubectl get clusterspiffeids.spire.spiffe.io spire-mgmt-spire-pod-label -o yaml | yq '.spec.dnsNameTemplates')
  if [[ $value != $expected ]]; then
    echo "Error: Did not find expected DNS name templates value: expected $expected, actual $value"
    return 1
  fi
}

function delete() {
  ./cofidectl attestation-policy-binding del --trust-zone $TRUST_ZONE --attestation-policy namespace
  ./cofidectl attestation-policy-binding del --trust-zone $TRUST_ZONE --attestation-policy pod-label
  ./cofidectl attestation-policy-binding del --trust-zone $TRUST_ZONE --attestation-policy static-namespace
  ./cofidectl attestation-policy del namespace
  ./cofidectl attestation-policy del pod-label
  ./cofidectl attestation-policy del static-namespace
  ./cofidectl cluster del $K8S_CLUSTER_NAME --trust-zone $TRUST_ZONE
  ./cofidectl trust-zone del $TRUST_ZONE
}

function main() {
  init $DATA_SOURCE_PLUGIN $PROVISION_PLUGIN
  configure
  up
  check_spire
  list_resources
  show_config
  show_status
  run_tests
  show_workload_status
  check_overridden_values
  check_dns_name_templates
  # Check for idempotence of up
  up
  down
  # Check for idempotence of down
  down
  delete
  check_delete
  echo "Success!"
}

main
