# This file provides common library functions for use by integration tests.

SPIRE_HELM_REPO_NAME=${SPIRE_HELM_REPO_NAME:-cofide}
SPIRE_HELM_REPO_URL=${SPIRE_HELM_REPO_URL:-https://charts.cofide.dev}
SPIRE_CHART_VERSION=${SPIRE_CHART_VERSION:-0.27.1-cofide.0}
SPIRE_CRDS_CHART_VERSION=${SPIRE_CRDS_CHART_VERSION:-0.5.0-cofide.1}
SPIRE_MGMT_NAMESPACE=${SPIRE_MGMT_NAMESPACE:-spire-mgmt}

function up() {
  helm repo add "$SPIRE_HELM_REPO_NAME" "$SPIRE_HELM_REPO_URL" --force-update
  helm repo update "$SPIRE_HELM_REPO_NAME"

  local trust_zones=("$@")
  if [[ ${#trust_zones[@]} -eq 0 ]]; then
    mapfile -t trust_zones < <(yq '.trust_zones[].name' cofide.yaml -r)
  fi

  local tmpfiles=()
  trap 'rm -f -- "${tmpfiles[@]}"' EXIT

  for tz in "${trust_zones[@]}"; do
    local tz_id context values_file
    tz_id=$(yq '.trust_zones[] | select(.name == "'"$tz"'") | .id' cofide.yaml -r)
    context=$(yq '.clusters[] | select(.trust_zone_id == "'"$tz_id"'") | .kubernetes_context' cofide.yaml -r)
    values_file=$(mktemp /tmp/cofide-values-XXXXXX.yaml)
    tmpfiles+=("$values_file")

    ./cofidectl trust-zone helm values "$tz" --output-file "$values_file"

    helm upgrade --install spire-crds "$SPIRE_HELM_REPO_NAME/spire-crds" \
      --kube-context "$context" \
      --namespace "$SPIRE_MGMT_NAMESPACE" \
      --create-namespace \
      --version "$SPIRE_CRDS_CHART_VERSION" \
      --wait

    helm upgrade --install spire "$SPIRE_HELM_REPO_NAME/spire" \
      --kube-context "$context" \
      --namespace "$SPIRE_MGMT_NAMESPACE" \
      --create-namespace \
      --version "$SPIRE_CHART_VERSION" \
      --values "$values_file" \
      --set "spire-server.federation.enabled=$(if [[ "$(yq "[.federations[] | select(.remote_trust_zone_id==\"$tz_id\")] | length" cofide.yaml)" -gt 0 ]]; then echo -n "true"; else echo -n "false"; fi)" \
      --wait

    rm -f -- "$values_file"
  done

  bootstrap_federation "${trust_zones[@]}"
}

# bootstrap_federation creates ClusterFederatedTrustDomain resources for each federation
# relationship defined in cofide.yaml. It must be called after the initial Helm install,
# once each SPIRE server is reachable and has generated its bundle.
#
# For the HTTPS_SPIFFE bundle endpoint profile used by default, the CFTD resource must
# include the remote trust zone's bundle (trustDomainBundle) so that SPIRE can bootstrap
# the TLS connection to the remote bundle endpoint. Without this, SPIRE cannot verify the
# remote certificate and federation will fail to establish.
function bootstrap_federation() {
  local fed_count
  fed_count=$(yq '.federations | length' cofide.yaml)
  if [[ -z "$fed_count" || "$fed_count" == "0" || "$fed_count" == "null" ]]; then
    return 0
  fi

  local trust_zones=("$@")
  if [[ ${#trust_zones[@]} -eq 0 ]]; then
    mapfile -t trust_zones < <(yq '.trust_zones[].name' cofide.yaml -r)
  fi

  # Collect external IP, SPIFFE bundle, kubernetes context, and trust domain for each trust zone.
  declare -A tz_ip tz_bundle tz_context tz_domain

  for tz in "${trust_zones[@]}"; do
    local tz_id context trust_domain ip bundle
    tz_id=$(yq '.trust_zones[] | select(.name == "'"$tz"'") | .id' cofide.yaml -r)
    context=$(yq '.clusters[] | select(.trust_zone_id == "'"$tz_id"'") | .kubernetes_context' cofide.yaml -r)
    trust_domain=$(yq '.trust_zones[] | select(.name == "'"$tz"'") | .trust_domain' cofide.yaml -r)

    # Wait for the SPIRE server StatefulSet to finish rolling out.
    kubectl --context "$context" rollout status statefulset/spire-server \
      -n spire-server --timeout=5m

    # Poll until the SPIRE server service has an external IP or hostname assigned.
    ip=""
    while [[ -z "$ip" ]]; do
      ip=$(kubectl --context "$context" get svc -n spire-server spire-server \
        -o "jsonpath={.status.loadBalancer.ingress[0].ip}" 2>/dev/null || true)
      if [[ -z "$ip" ]]; then
        ip=$(kubectl --context "$context" get svc -n spire-server spire-server \
          -o "jsonpath={.status.loadBalancer.ingress[0].hostname}" 2>/dev/null || true)
      fi
      [[ -z "$ip" ]] && sleep 2
    done

    # Retrieve the trust zone's bundle in SPIFFE JSON format.
    bundle=$(kubectl --context "$context" exec -n spire-server spire-server-0 -- \
      /opt/spire/bin/spire-server bundle show -format spiffe)

    tz_ip[$tz]="$ip"
    tz_bundle[$tz]="$bundle"
    tz_context[$tz]="$context"
    tz_domain[$tz]="$trust_domain"
  done

  # For each federation relationship, create a ClusterFederatedTrustDomain CR on the
  # source cluster that describes how to federate with the destination trust zone.
  while IFS=' ' read -r from_tz_id to_tz_id; do
    local from_tz to_tz from_context to_domain to_ip to_bundle
    from_tz=$(yq '.trust_zones[] | select(.id == "'"$from_tz_id"'") | .name' cofide.yaml -r)
    to_tz=$(yq '.trust_zones[] | select(.id == "'"$to_tz_id"'") | .name' cofide.yaml -r)
    from_context="${tz_context[$from_tz]}"
    to_domain="${tz_domain[$to_tz]}"
    to_ip="${tz_ip[$to_tz]}"
    to_bundle="${tz_bundle[$to_tz]}"

    kubectl --context "$from_context" apply -f - <<EOF
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterFederatedTrustDomain
metadata:
  name: ${to_domain}
spec:
  className: spire-mgmt-spire
  trustDomain: ${to_domain}
  bundleEndpointURL: https://${to_ip}:8443
  bundleEndpointProfile:
    type: https_spiffe
    endpointSPIFFEID: spiffe://${to_domain}/spire/server
  trustDomainBundle: |
$(echo "${to_bundle}" | sed 's/^/    /')
EOF
  done < <(yq '.federations[] | .trust_zone_id + " " + .remote_trust_zone_id' cofide.yaml -r)

  # Wait for the CFTD controller to push each remote bundle to the SPIRE server.
  # Once spire-server bundle list shows the remote trust domain, SPIRE has
  # federation configured and subsequent ClusterSPIFFEID entries with
  # federatesWith will be created successfully on first attempt.
  while IFS=' ' read -r from_tz_id to_tz_id; do
    local from_tz to_tz from_context to_domain
    from_tz=$(yq '.trust_zones[] | select(.id == "'"$from_tz_id"'") | .name' cofide.yaml -r)
    to_tz=$(yq '.trust_zones[] | select(.id == "'"$to_tz_id"'") | .name' cofide.yaml -r)
    from_context="${tz_context[$from_tz]}"
    to_domain="${tz_domain[$to_tz]}"

    local timeout=120 elapsed=0
    echo "Waiting for SPIRE on ${from_tz} to have bundle for spiffe://${to_domain}..."
    until kubectl --context "$from_context" exec -n spire-server spire-server-0 -- \
      /opt/spire/bin/spire-server bundle list -id "spiffe://${to_domain}" \
      2>/dev/null; do
      if [[ $elapsed -ge $timeout ]]; then
        echo "Timed out waiting for bundle spiffe://${to_domain} on ${from_tz}"
        return 1
      fi
      sleep 5
      elapsed=$((elapsed + 5))
    done
    echo "Bundle for spiffe://${to_domain} is available on ${from_tz}."
  done < <(yq '.federations[] | .trust_zone_id + " " + .remote_trust_zone_id' cofide.yaml -r)
}

function down() {
  local trust_zones=("$@")
  if [[ ${#trust_zones[@]} -eq 0 ]]; then
    mapfile -t trust_zones < <(yq '.trust_zones[].name' cofide.yaml -r)
  fi

  for tz in "${trust_zones[@]}"; do
    local tz_id context
    tz_id=$(yq '.trust_zones[] | select(.name == "'"$tz"'") | .id' cofide.yaml -r)
    context=$(yq '.clusters[] | select(.trust_zone_id == "'"$tz_id"'") | .kubernetes_context' cofide.yaml -r)

    helm uninstall spire \
      --kube-context "$context" \
      --namespace "$SPIRE_MGMT_NAMESPACE" \
      || true

    kubectl --context "$context" delete clusterfederatedtrustdomains \
      --all --ignore-not-found=true \
      || true
    kubectl --context "$context" delete clusterspiffeids.spire.spiffe.io \
      --all --ignore-not-found=true \
      || true
    kubectl --context "$context" delete clusterstaticentries.spire.spiffe.io \
      --all --ignore-not-found=true \
      || true

    helm uninstall spire-crds \
      --kube-context "$context" \
      --namespace "$SPIRE_MGMT_NAMESPACE" \
      || true
  done
}

function init() {
  local data_source_plugin=${1:-}
  local provision_plugin=${2:-}
  rm -f cofide.yaml
  local args=()
  if [[ -n "$data_source_plugin" ]]; then
    args+=("--data-source-plugin" "$data_source_plugin")
  fi
  if [[ -n "$provision_plugin" ]]; then
    args+=("--provision-plugin" "$provision_plugin")
  fi
  ./cofidectl init "${args[@]}"
}

function list_resources() {
  ./cofidectl trust-zone list
  ./cofidectl cluster list
  ./cofidectl attestation-policy list
  ./cofidectl attestation-policy-binding list
}

function show_config() {
  cat cofide.yaml
}

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

function run_ping_pong_test() {
  local client_context=${1:?Client Kubernetes context}
  local client_spiffe_ids="${2:?Client SPIFFE IDs}"
  local server_context=${3:?Server Kubernetes context}
  just -f demos/Justfile prompt_namespace=no deploy-ping-pong $client_context "$client_spiffe_ids" $server_context
  kubectl --context $client_context wait -n demo --for=condition=Available --timeout 60s deployments/ping-pong-client
  if ! wait_for_pong $client_context; then
    echo "Timed out waiting for pong from server"
    echo "Client logs:"
    kubectl --context $client_context logs -n demo deployments/ping-pong-client
    echo "Server logs:"
    kubectl --context $server_context logs -n demo deployments/ping-pong-server
    exit 1
  fi
}

function wait_for_pong() {
  local context=${1:?Kubernetes context}
  for i in $(seq 30); do
    if kubectl --context $context logs -n demo deployments/ping-pong-client | grep '\.\.\.pong'; then
      return 0
    fi
    sleep 2
  done
  return 1
}
