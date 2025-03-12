#!/bin/bash

set -euxo pipefail

NAMESPACE="$1"
CLIENT_CTX="$2"
SERVER_CTX="${3:-$CLIENT_CTX}"

COFIDE_DEMOS_BRANCH="https://raw.githubusercontent.com/cofide/cofide-demos/refs/heads/jsnctl/add-aws-oidc-workloads"

SERVER_MANIFEST="$COFIDE_DEMOS_BRANCH/workloads/ping-pong/ping-pong-server/deploy.yaml"
export PING_PONG_SERVER_SERVICE_PORT=8443
export IMAGE_TAG=v0.1.2-rc4
echo "Deploying pong server to: $SERVER_CTX"
if [[ $SERVER_CTX == kind-* ]]; then
    export KIND_CLUSTER_NAME="${SERVER_CTX#kind-}"
fi
if ! curl $SERVER_MANIFEST | envsubst | kubectl apply -n "$NAMESPACE" --context "$SERVER_CTX" -f -; then
    echo "Error: Server deployment failed" >&2
    exit 1
fi
echo "Server deployment complete"
if [[ "$CLIENT_CTX" == kind-* ]]; then
    export KIND_CLUSTER_NAME="${CLIENT_CTX#kind-}"
fi

echo "Deploying ping client to: $CLIENT_CTX"
if [[ "$SERVER_CTX" != "$CLIENT_CTX" ]]; then
    echo "Discovering server IP..."
    export PING_PONG_SERVER_SERVICE_HOST=$(kubectl --context "$SERVER_CTX" wait --for=jsonpath="{.status.loadBalancer.ingress[0].ip}" service/ping-pong-server -n $NAMESPACE --timeout=60s > /dev/null 2>&1 \
        && kubectl --context "$SERVER_CTX" get service ping-pong-server -n $NAMESPACE -o "jsonpath={.status.loadBalancer.ingress[0].ip}")
    echo "Server is $PING_PONG_SERVER_SERVICE_HOST"
else
    export PING_PONG_SERVER_SERVICE_HOST=ping-pong-server
fi
export PING_PONG_SERVER_SERVICE_PORT=8443
CLIENT_MANIFEST="$COFIDE_DEMOS_BRANCH/workloads/ping-pong/ping-pong-client/deploy.yaml"
if ! curl $CLIENT_MANIFEST | envsubst | ko resolve -f - | kubectl apply --context "$CLIENT_CTX" -n "$NAMESPACE" -f -; then
    echo "Error: client deployment failed" >&2
    exit 1
fi
echo "Client deployment complete"
