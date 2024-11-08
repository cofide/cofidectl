#!/bin/bash

# This script deploys 2 local Kubernetes clusters using Kind (https://kind.sigs.k8s.io).

set -euxo pipefail

K8S_CLUSTER_1_NAME=${K8S_CLUSTER_1_NAME:-local1}
K8S_CLUSTER_2_NAME=${K8S_CLUSTER_2_NAME:-local2}

function create_kind_cluster() {
  parent_dir=$(dirname $(dirname $BASH_SOURCE))
  export K8S_CLUSTER_NAME=$1
  $parent_dir/create-kind-cluster.sh
}

function main() {
  create_kind_cluster $K8S_CLUSTER_1_NAME
  create_kind_cluster $K8S_CLUSTER_2_NAME
}

main
