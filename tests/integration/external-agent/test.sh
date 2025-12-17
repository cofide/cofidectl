#!/bin/bash

# This script deploys a single trust zone with an external SPIRE agent, runs some basic tests against it, then tears it down.
# The trust zone has two static attestation policies for the ping pong client and server workloads.

set -euxo pipefail

source $(dirname $(dirname $BASH_SOURCE))/lib.sh

TEST_DIR=$(dirname $BASH_SOURCE)
CONF_DIR=$TEST_DIR/conf

DATA_SOURCE_PLUGIN=${DATA_SOURCE_PLUGIN:-}
PROVISION_PLUGIN=${PROVISION_PLUGIN:-}

K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME:-local1}
K8S_CLUSTER_CONTEXT=${K8S_CLUSTER_CONTEXT:-kind-$K8S_CLUSTER_NAME}

TRUST_ZONE=${TRUST_ZONE:-tz1}
TRUST_DOMAIN=${TRUST_DOMAIN:-td1}

AGENT_ID_PATH=test-agent
AGENT_ID=spiffe://$TRUST_DOMAIN/$AGENT_ID_PATH
AGENT_NAME=$TRUST_DOMAIN-spire-agent

PING_PONG_CLIENT_ID_PATH=app/ping-pong-client
PING_PONG_CLIENT_ID=spiffe://$TRUST_DOMAIN/$PING_PONG_CLIENT_ID_PATH
PING_PONG_CLIENT_NAME=$TRUST_DOMAIN-ping-pong-client

PING_PONG_SERVER_ID_PATH=app/ping-pong-server
PING_PONG_SERVER_ID=spiffe://$TRUST_DOMAIN/$PING_PONG_SERVER_ID_PATH
PING_PONG_SERVER_NAME=$TRUST_DOMAIN-ping-pong-server
PING_PONG_SERVER_DNS_NAME=ping-pong-server.example.org

function configure() {
  ./cofidectl trust-zone add $TRUST_ZONE \
    --trust-domain $TRUST_DOMAIN \
    --no-cluster
  ./cofidectl cluster add $K8S_CLUSTER_NAME \
    --trust-zone $TRUST_ZONE \
    --kubernetes-context $K8S_CLUSTER_CONTEXT \
    --profile kubernetes
  # A workload entry for the ping pong client.
  ./cofidectl attestation-policy add static \
    --name ping-pong-client \
    --spiffe-id-path $PING_PONG_CLIENT_ID_PATH \
    --parent-id-path $AGENT_ID_PATH \
    --selectors docker:label:app:ping-pong-client
  ./cofidectl attestation-policy-binding add \
    --trust-zone $TRUST_ZONE \
    --attestation-policy ping-pong-client
  # A workload entry for the ping pong server.
  ./cofidectl attestation-policy add static \
    --name ping-pong-server \
    --spiffe-id-path $PING_PONG_SERVER_ID_PATH \
    --parent-id-path $AGENT_ID_PATH \
    --selectors docker:label:app:ping-pong-server \
    --dns-names $PING_PONG_SERVER_DNS_NAME
  ./cofidectl attestation-policy-binding add \
    --trust-zone $TRUST_ZONE \
    --attestation-policy ping-pong-server
  override_helm_values
}

function configure_spire_agent() {
  mkdir -p $CONF_DIR
  kubectl --context $K8S_CLUSTER_CONTEXT -n spire-server \
    exec spire-server-0 -- \
    spire-server bundle show > $CONF_DIR/bundle.crt
  local server_address=$(kubectl --context $K8S_CLUSTER_CONTEXT -n spire-server \
    get svc spire-server \
    -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
  local join_token=$(kubectl --context $K8S_CLUSTER_CONTEXT -n spire-server \
    exec spire-server-0 -- \
    spire-server token generate -spiffeID $AGENT_ID -output json \
    | jq -er .value)
  TRUST_DOMAIN=$TRUST_DOMAIN \
    SERVER_ADDRESS=$server_address \
    JOIN_TOKEN=$join_token \
    envsubst <$TEST_DIR/templates/agent.conf >$CONF_DIR/agent.conf
}

function deploy_spire_agent() {
  docker rm -f $AGENT_NAME || true
  # The kind network is required to access the SPIRE server's load balancer service.
  # Host PID namespace is required to attest processes outside the container.
  # Docker socket mount is required for the Docker workload attestor.
  docker run --detach \
    --name $AGENT_NAME \
    --network kind \
    --pid host \
    --restart unless-stopped \
    --volume $(realpath $CONF_DIR):/opt/spire/conf \
    --volume /var/run/docker.sock:/var/run/docker.sock:ro \
    ghcr.io/spiffe/spire-agent:1.12.6 \
    -config /opt/spire/conf/agent.conf
}

function override_helm_values() {
  # Enable the join token node attestor.
  ./cofidectl trust-zone helm override $TRUST_ZONE --input-file - <<EOF
spire-server:
  nodeAttestor:
    joinToken:
      enabled: true
EOF
}

function check_spire() {
  check_spire_server $K8S_CLUSTER_CONTEXT
  check_spire_agents $K8S_CLUSTER_CONTEXT
  check_spire_csi_driver $K8S_CLUSTER_CONTEXT
  check_external_spire_agent
}

function check_external_spire_agent() {
  for i in $(seq 60); do
    status=$(docker ps --filter name=$AGENT_NAME --format '{{ .Status }}')
    if [[ $status =~ ^Up ]]; then
      return 0
    fi
    if [[ -z $status ]]; then
      echo "External SPIRE agent container not found"
      return 1
    fi
    sleep 2
  done
  echo "Timed out waiting for external SPIRE agent"
  docker logs $AGENT_NAME
  return 1
}

function show_status() {
  ./cofidectl trust-zone status $TRUST_ZONE
}

function run_tests() {
  run_ping_pong_docker_test
  check_ping_pong_server_dns_name
}

function run_ping_pong_docker_test() {
  local workload_api_path="$(realpath $CONF_DIR/spire-agent.sock)"
  just -f demos/Justfile deploy-ping-pong-docker \
    $PING_PONG_CLIENT_NAME \
    $PING_PONG_CLIENT_ID \
    $PING_PONG_SERVER_NAME \
    $workload_api_path
  if ! wait_for_pong; then
    echo "Timed out waiting for pong from server"
    echo "Client logs:"
    docker logs $PING_PONG_CLIENT_NAME
    echo "Server logs:"
    docker logs $PING_PONG_SERVER_NAME
    exit 1
  fi
}

function check_ping_pong_server_dns_name() {
  # Connect to the ping pong server and check that it has the DNS SAN requested in the attestation policy.
  # The first command exits non-zero due to the server requiring a client cert, but we can ignore this.
  local server_cert=$(openssl s_client -connect localhost:8443 -showcerts 2</dev/null \
    | openssl x509 -noout -text \
    || true)
  echo "$server_cert" | grep "\bDNS:$PING_PONG_SERVER_DNS_NAME\b"
}

function wait_for_pong() {
  for i in $(seq 30); do
    if docker logs $PING_PONG_CLIENT_NAME 2>&1 | grep '\.\.\.pong'; then
      return 0
    fi
    sleep 2
  done
  return 1
}

function tear_down_workloads() {
  docker rm -f $PING_PONG_CLIENT_NAME || true
  docker rm -f $PING_PONG_SERVER_NAME || true
}

function tear_down_spire_agent() {
  docker rm -f $AGENT_NAME || true
}

function delete() {
  ./cofidectl attestation-policy-binding del \
    --trust-zone $TRUST_ZONE \
    --attestation-policy ping-pong-client
  ./cofidectl attestation-policy-binding del \
    --trust-zone $TRUST_ZONE \
    --attestation-policy ping-pong-server
  ./cofidectl attestation-policy del ping-pong-client
  ./cofidectl attestation-policy del ping-pong-server
  ./cofidectl cluster del $K8S_CLUSTER_NAME --trust-zone $TRUST_ZONE
  ./cofidectl trust-zone del $TRUST_ZONE
}

function main() {
  init $DATA_SOURCE_PLUGIN $PROVISION_PLUGIN
  configure
  up $TRUST_ZONE
  configure_spire_agent
  deploy_spire_agent
  check_spire
  list_resources
  show_config
  show_status
  run_tests
  tear_down_workloads
  tear_down_spire_agent
  down $TRUST_ZONE
  delete
  check_delete
  echo "Success!"
}

main
