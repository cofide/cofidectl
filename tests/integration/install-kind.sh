#!/bin/bash

# This script installs Kind (https://kind.sigs.k8s.io) for test and development of cofidectl with local Kubernetes clusters.

set -euxo pipefail

function prechecks() {
  if ! type apt; then
    echo "Only Ubuntu is supported"
  fi
}

function install_package_deps() {
  sudo apt update
  sudo apt install -y git
}

function install_docker() {
  # https://docs.docker.com/engine/install/ubuntu/
  # Add Docker's official GPG key:
  sudo apt-get update
  sudo apt-get install -y ca-certificates curl
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  sudo chmod a+r /etc/apt/keyrings/docker.asc

  # Add the repository to Apt sources:
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
}

function install_kind () {
  # https://kind.sigs.k8s.io/docs/user/quick-start#installing-from-release-binaries
  # For AMD64 / x86_64
  [ "$(uname -m)" = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.24.0/kind-linux-amd64
  # For ARM64
  [ "$(uname -m)" = aarch64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.24.0/kind-linux-arm64
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
}

function set_inotify_limits_for_kind() {
  # https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files
  cat << EOF | sudo tee /etc/sysctl.d/10-kind.conf
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 512
EOF
}

function ensure_kind_network() {
  # cloud-provider-kind requires the kind network to exist, which gets created the first time a kind cluster is created.
  if ! sudo docker network inspect kind &>/dev/null; then
    sudo kind create cluster
    sudo kind delete cluster
  fi
}

function run_cloud_provider_kind() {
docker run -d \
    --network kind \
    --restart unless-stopped \
    --privileged \
    -v /var/run/docker.sock:/var/run/docker.sock \
    registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.5.0
}

function check_non_root_docker_access() {
  if ! docker ps &>/dev/null; then
    echo "You may need to log out and in again to obtain non-root access to Docker"
  fi
}

function main() {
  prechecks
  install_package_deps
  install_docker
  install_kind
  set_inotify_limits_for_kind
  ensure_kind_network
  run_cloud_provider_kind
  echo "Success!"
  check_non_root_docker_access
}

main
