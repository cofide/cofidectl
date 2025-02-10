#!/bin/bash

# This script deploys a local Kubernetes cluster using Kind (https://kind.sigs.k8s.io).

set -euxo pipefail

DELETE_EXISTING_KIND_CLUSTER=${DELETE_EXISTING_KIND_CLUSTER:-true}

K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME:-local1}

CLOUD_PROVIDER_KIND_CONTAINER_NAME=cloud-provider-kind

function delete_kind_cluster() {
  cluster=$(kind get clusters | egrep "\b${K8S_CLUSTER_NAME}\b" || true)
  if [[ -n $cluster ]]; then
    kind delete cluster -n $K8S_CLUSTER_NAME
  fi
}

function create_kind_cluster() {
  kind create cluster -n $K8S_CLUSTER_NAME
}

function restart_cloud_provider_kind() {
  # Workaround: cloud-provider-kind often stops working when a kind cluster is created. Restart it.
  if [[ $(docker ps -q --filter "name=$CLOUD_PROVIDER_KIND_CONTAINER_NAME") != "" ]]; then
    docker restart $CLOUD_PROVIDER_KIND_CONTAINER_NAME
  fi
}

function main() {
  if $DELETE_EXISTING_KIND_CLUSTER; then
    delete_kind_cluster
  fi
  create_kind_cluster
  restart_cloud_provider_kind
  echo "Success!"
}

main
