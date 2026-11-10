setup() {
  load 'test_helper'
  _common_setup
}

jenkins_ready() {
  local completed
  completed=$(${KUBECTL} get jenkins jenkins -o jsonpath='{.status.userConfigurationCompletedTime}') || return 1
  [[ -n "${completed}" ]] || return 1
  ${KUBECTL} wait --for=condition=Ready pod/jenkins-jenkins --timeout=5s
}

pod_replaced() {
  local uid
  uid=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}') || return 1
  [[ -n "${uid}" && "${uid}" != "$1" ]]
}

assert_pod_unchanged() {
  local uid="$1"
  local cycle sample
  for cycle in 1 2 3; do
    run ${KUBECTL} annotate jenkins jenkins "reconciliation.test/reconcile=$(date +%s%N)" --overwrite
    assert_success
    for sample in 1 2 3 4 5; do
      sleep 2
      run ${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}{"|"}{.metadata.deletionTimestamp}'
      assert_success
      assert_output "${uid}|"
    done
  done
}

#bats test_tags=phase:helm,scenario:reconciliation-ignore
@test "4.0 Helm: preserve Jenkins for ignored metadata and volume changes" {
  if ! ${KUBECTL} get namespace "${DETIK_CLIENT_NAMESPACE}"; then
    run ${KUBECTL} create namespace "${DETIK_CLIENT_NAMESPACE}"
    assert_success
  fi
  run ${KUBECTL} apply -f config/crd/bases/jenkins.io_jenkins.yaml
  assert_success
  run ${KUBECTL} create configmap ignored-original --from-literal=value=original
  assert_success
  run ${KUBECTL} create configmap ignored-updated --from-literal=value=updated
  assert_success

  cat > "${BATS_TEST_TMPDIR}/values.yaml" <<'YAML'
jenkins:
  labels:
    reconciliation.test/managed: declared
    reconciliation.test/control: original
  annotations:
    reconciliation.test/managed: declared
  reconciliation:
    ignore:
      labels: [reconciliation.test/managed]
      annotations: [reconciliation.test/managed]
      volumes: [ignored-config]
  volumes:
    - name: backup
      persistentVolumeClaim:
        claimName: jenkins-backup
    - name: ignored-config
      configMap:
        name: ignored-original
  volumeMounts:
    - name: ignored-config
      mountPath: /var/ignored-config
      readOnly: true
  backup:
    makeBackupBeforePodDeletion: false
YAML

  run ${HELM} install reconciliation chart/jenkins-operator \
    -f "${BATS_TEST_TMPDIR}/values.yaml" \
    --set "namespace=${DETIK_CLIENT_NAMESPACE}" \
    --set "jenkins.namespace=${DETIK_CLIENT_NAMESPACE}" \
    --set "operator.image=${OPERATOR_IMAGE}" \
    --set jenkins.imagePullPolicy=IfNotPresent \
    --set jenkins.backup.image=quay.io/jenkins-kubernetes-operator/backup-pvc:e2e-test \
    --wait --timeout=5m
  assert_success

  run ${KUBECTL} get jenkins jenkins -o jsonpath='{.spec.reconciliation.ignoreWellKnownInjections}{"|"}{.spec.reconciliation.ignore.labels[0]}{"|"}{.spec.reconciliation.ignore.annotations[0]}{"|"}{.spec.reconciliation.ignore.volumes[0]}'
  assert_success
  assert_output 'true|reconciliation.test/managed|reconciliation.test/managed|ignored-config'

  run ${HELM} upgrade reconciliation chart/jenkins-operator --reuse-values \
    --set jenkins.reconciliation.ignoreWellKnownInjections=false --wait
  assert_success
  run ${KUBECTL} get jenkins jenkins -o jsonpath='{.spec.reconciliation.ignoreWellKnownInjections}'
  assert_success
  assert_output 'false'
  run ${HELM} upgrade reconciliation chart/jenkins-operator --reuse-values \
    --set jenkins.reconciliation.ignoreWellKnownInjections=true --wait
  assert_success

  run retry 120 5 jenkins_ready
  assert_success
  local uid
  uid=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.metadata.uid}')
  [[ -n "${uid}" ]]
  run ${KUBECTL} get pod jenkins-jenkins -o go-template='{{index .metadata.labels "reconciliation.test/managed"}}|{{index .metadata.annotations "reconciliation.test/managed"}}|{{range .spec.volumes}}{{if eq .name "ignored-config"}}{{.configMap.name}}{{end}}{{end}}'
  assert_success
  assert_output 'declared|declared|ignored-original'
  run ${KUBECTL} exec jenkins-jenkins -c jenkins-master -- cat /var/ignored-config/value
  assert_success
  assert_output 'original'

  run ${KUBECTL} label pod jenkins-jenkins reconciliation.test/managed=runtime --overwrite
  assert_success
  run ${KUBECTL} annotate pod jenkins-jenkins reconciliation.test/managed=runtime --overwrite
  assert_success
  assert_pod_unchanged "${uid}"
  run ${KUBECTL} get pod jenkins-jenkins -o go-template='{{index .metadata.labels "reconciliation.test/managed"}}|{{index .metadata.annotations "reconciliation.test/managed"}}'
  assert_success
  assert_output 'runtime|runtime'

  # Pod volume sources are immutable; change the declared source instead.
  run ${KUBECTL} patch jenkins jenkins --type=json \
    -p='[{"op":"replace","path":"/spec/master/volumes/1/configMap/name","value":"ignored-updated"}]'
  assert_success
  assert_pod_unchanged "${uid}"
  run ${KUBECTL} get jenkins jenkins -o jsonpath='{.spec.master.volumes[?(@.name=="ignored-config")].configMap.name}'
  assert_success
  assert_output 'ignored-updated'
  run ${KUBECTL} get pod jenkins-jenkins -o jsonpath='{.spec.volumes[?(@.name=="ignored-config")].configMap.name}'
  assert_success
  assert_output 'ignored-original'

  run ${KUBECTL} patch jenkins jenkins --type=merge \
    -p='{"spec":{"master":{"labels":{"reconciliation.test/control":"changed"}}}}'
  assert_success
  run retry 120 5 "pod_replaced ${uid}"
  assert_success
  run retry 120 5 jenkins_ready
  assert_success
  run ${KUBECTL} get pod jenkins-jenkins -o go-template='{{index .metadata.labels "reconciliation.test/control"}}|{{range .spec.volumes}}{{if eq .name "ignored-config"}}{{.configMap.name}}{{end}}{{end}}'
  assert_success
  assert_output 'changed|ignored-updated'
  run ${KUBECTL} exec jenkins-jenkins -c jenkins-master -- cat /var/ignored-config/value
  assert_success
  assert_output 'updated'
}

#bats test_tags=phase:helm,scenario:reconciliation-ignore
@test "4.1 Helm: clean" {
  run ${HELM} uninstall reconciliation --wait
  assert_success
  run ${KUBECTL} delete configmap ignored-original ignored-updated --ignore-not-found
  assert_success
}
