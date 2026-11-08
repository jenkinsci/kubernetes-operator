mesh_jenkins_ready() {
  local completed
  completed=$(${KUBECTL} get jenkins jenkins -o jsonpath='{.status.userConfigurationCompletedTime}') || return 1
  [[ -n "$completed" ]] || return 1
  ${KUBECTL} wait --for=condition=Ready pod/jenkins-jenkins --timeout=5s
}

mesh_pod_replaced() {
  local current
  current=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}') || return 1
  [[ -n "$current" && "$current" != "$1" ]]
}

mesh_assert_stable() {
  local uid="$1" cycle sample
  for cycle in 1 2 3; do
    run ${KUBECTL} annotate jenkins jenkins "mesh.test/reconcile=$(date +%s%N)" --overwrite
    assert_success
    for sample in 1 2 3 4 5; do
      sleep 2
      run ${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}{"|"}{.metadata.deletionTimestamp}'
      assert_success
      assert_output "${uid}|"
    done
  done
}

mesh_scenario() {
  local mesh="$1" mode="$2"
  local namespace="${DETIK_CLIENT_NAMESPACE}-${mesh}-${mode}"
  local KUBECTL="kubectl --context=${CONTEXT} -n ${namespace}"
  local HELM="helm --kube-context=${CONTEXT} -n ${namespace}"
  local native=false proxy=istio-proxy
  [[ "$mode" != native ]] || native=true
  run ${KUBECTL} create namespace "$namespace"
  assert_success
  run ${KUBECTL} apply -f config/crd/bases/jenkins.io_jenkins.yaml
  assert_success
  cat > "$BATS_TEST_TMPDIR/mesh-values.yaml" <<'YAML'
jenkins:
  volumes: []
  backup:
    enabled: false
    pvc:
      enabled: false
YAML
  if [[ "$mesh" == istio ]]; then
    cat >> "$BATS_TEST_TMPDIR/mesh-values.yaml" <<YAML
  labels:
    sidecar.istio.io/inject: "true"
  annotations:
    sidecar.istio.io/rewriteAppHTTPProbers: "false"
    sidecar.istio.io/nativeSidecar: "$native"
YAML
  else
    proxy=linkerd-proxy
    cat >> "$BATS_TEST_TMPDIR/mesh-values.yaml" <<YAML
  annotations:
    linkerd.io/inject: enabled
    config.linkerd.io/proxy-enable-native-sidecar: "$native"
YAML
  fi
  run ${HELM} install mesh chart/jenkins-operator -f "$BATS_TEST_TMPDIR/mesh-values.yaml" \
    --set "namespace=$namespace" --set "jenkins.namespace=$namespace" \
    --set "operator.image=${OPERATOR_IMAGE}" --set jenkins.imagePullPolicy=IfNotPresent --wait --timeout=5m
  assert_success
  run retry 150 5 mesh_jenkins_ready
  assert_success
  local field=containers uid probe
  [[ "$native" != true ]] || field=initContainers
  run ${KUBECTL} get pod jenkins-jenkins -o "jsonpath={.spec.${field}[?(@.name==\"${proxy}\")].name}"
  assert_success
  assert_output "$proxy"
  if [[ "$native" == true ]]; then
    run ${KUBECTL} get pod jenkins-jenkins -o "jsonpath={.spec.initContainers[?(@.name==\"${proxy}\")].restartPolicy}"
    assert_success
    assert_output Always
  fi
  probe=$(${KUBECTL} get jenkins jenkins -o jsonpath='{.spec.master.containers[0].readinessProbe.httpGet.path}{"|"}{.spec.master.containers[0].readinessProbe.httpGet.port}')
  run ${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.spec.containers[?(@.name=="jenkins-master")].readinessProbe.httpGet.path}{"|"}{.spec.containers[?(@.name=="jenkins-master")].readinessProbe.httpGet.port}'
  assert_success
  assert_output "$probe"
  uid=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}')
  [[ -n "$uid" ]]
  mesh_assert_stable "$uid"

  run ${KUBECTL} patch jenkins jenkins --type=json -p='[{"op":"replace","path":"/spec/master/containers/0/readinessProbe/timeoutSeconds","value":11}]'
  assert_success
  run retry 120 5 "mesh_pod_replaced $uid"
  assert_success
  run retry 150 5 mesh_jenkins_ready
  assert_success
  run ${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.spec.containers[?(@.name=="jenkins-master")].readinessProbe.timeoutSeconds}'
  assert_success
  assert_output 11
  uid=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}')
  mesh_assert_stable "$uid"
  run ${HELM} uninstall mesh --wait
  assert_success
  run ${KUBECTL} delete namespace "$namespace" --wait=true --timeout=2m
  assert_success
}

mesh_cleanup() {
  local mesh="$1" mode namespace
  for mode in regular native; do
    namespace="${DETIK_CLIENT_NAMESPACE}-${mesh}-${mode}"
    run helm --kube-context="${CONTEXT}" -n "$namespace" uninstall mesh --wait --ignore-not-found
    assert_success
    run kubectl --context="${CONTEXT}" delete namespace "$namespace" --ignore-not-found --wait=true --timeout=2m
    assert_success
  done
  run make mesh-clean "MESH=$mesh"
  assert_success
}
