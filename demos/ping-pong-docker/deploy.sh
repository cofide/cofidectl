#!/bin/bash

# This script deploys ping pong client and server workloads in Docker containers and waits for
# them to be up.

set -euxo pipefail

CLIENT_NAME="${1?Client name is required}"
CLIENT_SPIFFE_IDS="${2?Client SPIFFE IDs is required}"
SERVER_NAME="${3?Server name is required}"
WORKLOAD_API_PATH="${4?Workload API path is required}"

export IMAGE_TAG=v0.4.3

# Wait for a Docker container to be up.
function wait_until_up() {
  local name=${1?Container name}
  for i in $(seq 60); do
    status=$(docker ps --filter name=$name --format '{{ .Status }}')
    if [[ $status =~ ^Up ]]; then
      return 0
    fi
    if [[ -z $status ]]; then
      echo "Docker container $name not found"
      return 1
    fi
    sleep 2
  done
  echo "Timed out waiting for Docker container $name to be up"
  return 1
}

function deploy_server() {
  docker rm -f $SERVER_NAME || true
  docker run --detach \
    --name $SERVER_NAME \
    --label app=ping-pong-server \
    --network kind \
    --publish 8443:8443 \
    --restart unless-stopped \
    --env "CLIENT_SPIFFE_IDS=$CLIENT_SPIFFE_IDS" \
    --volume $WORKLOAD_API_PATH:/spiffe-workload-api/spire-agent.sock \
    ghcr.io/cofide/cofide-demos/ping-pong-server:$IMAGE_TAG
  wait_until_up $SERVER_NAME
  echo "Server deployment complete"
}

function deploy_client() {
  docker rm -f $CLIENT_NAME || true
  docker run --detach \
    --name $CLIENT_NAME \
    --label app=ping-pong-client \
    --network kind \
    --restart unless-stopped \
    --env PING_PONG_SERVICE_HOST=$SERVER_NAME \
    --env PING_PONG_SERVICE_PORT=8443 \
    --volume $WORKLOAD_API_PATH:/spiffe-workload-api/spire-agent.sock \
    ghcr.io/cofide/cofide-demos/ping-pong-client:$IMAGE_TAG
  wait_until_up $CLIENT_NAME
  echo "Client deployment complete"
}

function main() {
  deploy_server
  deploy_client
}

main
