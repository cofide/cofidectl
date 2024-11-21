#!/bin/bash

# This script deploys multiple local Kubernetes clusters using Kind (https://kind.sigs.k8s.io).

set -euxo pipefail

NUM_K8S_CLUSTERS=${1:?Number of kind clusters to create}
K8S_CLUSTER_NAME_PREFIX=${K8S_CLUSTER_NAME_PREFIX:-local}

function create_kind_cluster() {
  suffix=$1
  parent_dir=$(dirname $BASH_SOURCE)
  export K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME_PREFIX}${suffix}
  $parent_dir/create-kind-cluster.sh
}

function main() {
  for i in $(seq $NUM_K8S_CLUSTERS); do
    create_kind_cluster $i
  done
}

main
