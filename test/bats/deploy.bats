setup() {
    load 'test_helper'
    _common_setup
}

#bats test_tags=phase:setup
@test "1.0 Create namespace" {
    ${KUBECTL} get ns ${DETIK_CLIENT_NAMESPACE} && skip "Namespace ${DETIK_CLIENT_NAMESPACE} already exists"
    run ${KUBECTL} create ns ${DETIK_CLIENT_NAMESPACE}
    assert_success
}

#bats test_tags=phase:helm
@test "1.1 Vanilla install helm chart" {
    ${HELM} status default && skip "Helm release 'default' already exists"
    run ${HELM} install default \
      --set namespace=${DETIK_CLIENT_NAMESPACE} \
      --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
      --set operator.image=${OPERATOR_IMAGE} \
      chart/jenkins-operator
    assert_success
    assert ${HELM} status default
}

@test "1.2 Temp check logs" {
   # Temporary debug logs will create the appriate check
   sleep 240
   ${KUBECTL} logs -l app=jenkins-operator,jenkins-cr=jenkins --tail=100000
   ${KUBECTL} logs -l app=jenkins-operator,jenkins-cr=jenkins --tail=100000 -p || true
   ${KUBECTL} logs -l app.kubernetes.io/name=jenkins-operator --tail=100000
   ${KUBECTL} logs -l app.kubernetes.io/name=jenkins-operator --tail=100000 -p || true
}
