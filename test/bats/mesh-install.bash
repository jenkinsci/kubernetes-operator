#!/usr/bin/env bash
set -euo pipefail

action="$1"
mesh="$2"
istio_version="$3"
linkerd_version="$4"
context="$5"
mesh_tmp=$(mktemp -d)
trap 'rm -rf "$mesh_tmp"' EXIT

helm_mesh() {
  bin/helm --kube-context="$context" --repository-config="$mesh_tmp/repositories.yaml" --repository-cache="$mesh_tmp/cache" "$@"
}

case "$mesh" in
  istio) namespace=istio-system; releases=(bats-istiod bats-istio-base); crd_release=bats-istio-base ;;
  linkerd) namespace=linkerd; releases=(bats-linkerd bats-linkerd-crds); crd_release=bats-linkerd-crds ;;
  *) echo "Unknown mesh: $mesh" >&2; exit 1 ;;
esac

if [[ "$action" == clean ]]; then
  kubectl --context="$context" get crds -o "go-template={{range .items}}{{\$crd := .}}{{with .metadata.annotations}}{{if eq (index . \"meta.helm.sh/release-name\") \"$crd_release\"}}{{\$crd.metadata.name}}{{\"\\n\"}}{{end}}{{end}}{{end}}" > "$mesh_tmp/crds"
  mapfile -t crds < "$mesh_tmp/crds"
  helm_mesh uninstall "${releases[@]}" --namespace="$namespace" --wait --ignore-not-found
  if [[ ${#crds[@]} -gt 0 ]]; then
    kubectl --context="$context" delete crd "${crds[@]}" --ignore-not-found
  fi
  kubectl --context="$context" delete namespace "$namespace" --wait=true --timeout=2m --ignore-not-found
  exit
fi
[[ "$action" == install ]]

if [[ "$mesh" == istio ]]; then
  helm_mesh install bats-istio-base base --repo https://istio-release.storage.googleapis.com/charts --version "$istio_version" --namespace "$namespace" --create-namespace
  helm_mesh install bats-istiod istiod --repo https://istio-release.storage.googleapis.com/charts --version "$istio_version" --namespace "$namespace" --wait --timeout=5m
else
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -keyout "$mesh_tmp/ca.key" -out "$mesh_tmp/ca.crt" -days 2 -subj /CN=root.linkerd.cluster.local -addext basicConstraints=critical,CA:TRUE -addext keyUsage=critical,keyCertSign,cRLSign >/dev/null 2>&1
  openssl req -new -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -keyout "$mesh_tmp/issuer.key" -out "$mesh_tmp/issuer.csr" -subj /CN=identity.linkerd.cluster.local >/dev/null 2>&1
  printf 'basicConstraints=critical,CA:TRUE\nkeyUsage=critical,keyCertSign,cRLSign\n' > "$mesh_tmp/issuer.ext"
  openssl x509 -req -in "$mesh_tmp/issuer.csr" -CA "$mesh_tmp/ca.crt" -CAkey "$mesh_tmp/ca.key" -CAcreateserial -out "$mesh_tmp/issuer.crt" -days 2 -extfile "$mesh_tmp/issuer.ext" >/dev/null 2>&1
  helm_mesh install bats-linkerd-crds linkerd-crds --repo https://helm.linkerd.io/edge --version "$linkerd_version" --namespace "$namespace" --create-namespace --set installGatewayAPI=true
  helm_mesh install bats-linkerd linkerd-control-plane --repo https://helm.linkerd.io/edge --version "$linkerd_version" --namespace "$namespace" --set-file "identityTrustAnchorsPEM=$mesh_tmp/ca.crt" --set-file "identity.issuer.tls.crtPEM=$mesh_tmp/issuer.crt" --set-file "identity.issuer.tls.keyPEM=$mesh_tmp/issuer.key" --wait --timeout=5m
fi
