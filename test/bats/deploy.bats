setup() {
    load 'test_helper'
    _common_setup
}

@test "1.0 Create namespace" {
    ${KUBECTL} get ns ${DETIK_CLIENT_NAMESPACE} && skip "Namespace ${DETIK_CLIENT_NAMESPACE} already exists"
    run ${KUBECTL} create ns ${DETIK_CLIENT_NAMESPACE}
    assert_success
}

@test "1.1 Vanilla install helm" {
    ${HELM} status default && skip "Helm release 'default' already exists"
    run ${HELM} install default \
      --set namespace=${DETIK_CLIENT_NAMESPACE} \
      --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
      --set operator.image=${OPERATOR_IMAGE} \
      chart/jenkins-operator
    assert_success
    assert ${HELM} status default
}
