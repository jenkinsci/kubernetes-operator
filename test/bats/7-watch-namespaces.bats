setup() {
  load 'test_helper'
  _common_setup
}

namespace_jenkins_ready() {
  local completed
  completed=$(${KUBECTL} -n "$1" get jenkins jenkins -o jsonpath='{.status.userConfigurationCompletedTime}') || return 1
  [[ -n "${completed}" ]] || return 1
  ${KUBECTL} -n "$1" wait --for=condition=Ready pod/jenkins-jenkins --timeout=5s
}

namespace_jenkins() {
  local namespace="$1"
  run ${HELM} template namespace-watch chart/jenkins-operator \
    --show-only templates/jenkins.yaml \
    --set "jenkins.namespace=${namespace}" \
    --set jenkins.backup.enabled=false \
    --set jenkins.imagePullPolicy=IfNotPresent \
    --set-json 'jenkins.volumes=[]'
  assert_success
  printf '%s\n' "${output}" > "${BATS_TEST_TMPDIR}/jenkins.yaml"
  run ${KUBECTL} -n "${namespace}" apply -f "${BATS_TEST_TMPDIR}/jenkins.yaml"
  assert_success
  run retry 60 5 "namespace_jenkins_ready '${namespace}'"
  assert_success
  run ${KUBECTL} -n "${namespace}" delete jenkins jenkins --wait --timeout=60s
  assert_success
  run ${KUBECTL} -n "${namespace}" wait --for=delete pod/jenkins-jenkins --timeout=120s
  assert_success
}

#bats test_tags=phase:helm,scenario:watch-namespaces
@test "7.0 Helm: align watch scope and permissions across namespaces" {
  local namespace
  for namespace in "${DETIK_CLIENT_NAMESPACE}" "${DETIK_CLIENT_NAMESPACE}-target" "${DETIK_CLIENT_NAMESPACE}-other"; do
    if ! ${KUBECTL} get namespace "${namespace}"; then
      run ${KUBECTL} create namespace "${namespace}"
      assert_success
    fi
  done
  run ${KUBECTL} apply --server-side --force-conflicts -f chart/jenkins-operator/crds/jenkins-crd.yaml
  assert_success
  run ${KUBECTL} wait --for=condition=Established crd/jenkins.jenkins.io --timeout=60s
  assert_success

  local account="system:serviceaccount:${DETIK_CLIENT_NAMESPACE}:jenkins-operator"
  run ${HELM} install namespace-watch chart/jenkins-operator \
    --set jenkins.enabled=false \
    --set jenkins.backup.pvc.enabled=false \
    --set "jenkins.namespace=${DETIK_CLIENT_NAMESPACE}" \
    --set "operator.image=${OPERATOR_IMAGE}" \
    --wait --timeout=2m
  assert_success
  run ${KUBECTL} auth can-i list jenkins --as="${account}"
  assert_success
  assert_output yes

  run ${HELM} upgrade namespace-watch chart/jenkins-operator --reuse-values \
    --set-string operator.watchNamespace= --wait --timeout=2m
  assert_success
  run ${KUBECTL} auth can-i list jenkins --as="${account}"
  assert_success
  assert_output yes
  run ${KUBECTL} -n "${DETIK_CLIENT_NAMESPACE}-other" auth can-i list jenkins --as="${account}"
  assert_failure
  assert_output no

  run ${HELM} upgrade namespace-watch chart/jenkins-operator --reuse-values \
    --set "operator.watchNamespace=${DETIK_CLIENT_NAMESPACE}-target" --wait --timeout=2m
  assert_success
  run ${KUBECTL} -n "${DETIK_CLIENT_NAMESPACE}-target" auth can-i list jenkins --as="${account}"
  assert_success
  assert_output yes
  run ${KUBECTL} -n "${DETIK_CLIENT_NAMESPACE}-other" auth can-i list jenkins --as="${account}"
  assert_failure
  assert_output no
  run ${KUBECTL} rollout status deployment/namespace-watch-jenkins-operator --timeout=120s
  assert_success
  namespace_jenkins "${DETIK_CLIENT_NAMESPACE}-target"

  run ${HELM} upgrade namespace-watch chart/jenkins-operator --reuse-values \
    --set operator.watchAllNamespaces=true --wait --timeout=2m
  assert_success
  run ${KUBECTL} -n "${DETIK_CLIENT_NAMESPACE}-other" auth can-i list jenkins --as="${account}"
  assert_success
  assert_output yes
  run ${KUBECTL} rollout status deployment/namespace-watch-jenkins-operator --timeout=120s
  assert_success
  run ${KUBECTL} get deployment namespace-watch-jenkins-operator -o 'jsonpath={.spec.template.spec.containers[0].env[?(@.name=="WATCH_NAMESPACE")].value}'
  assert_success
  assert_output ''
  namespace_jenkins "${DETIK_CLIENT_NAMESPACE}-other"
}

#bats test_tags=phase:helm,scenario:watch-namespaces
@test "7.1 Helm: clean namespace watching scenario" {
  run ${HELM} uninstall namespace-watch --wait --timeout=2m
  assert_success
  run ${KUBECTL} delete namespace "${DETIK_CLIENT_NAMESPACE}-target" "${DETIK_CLIENT_NAMESPACE}-other" --wait --timeout=2m
  assert_success
}
