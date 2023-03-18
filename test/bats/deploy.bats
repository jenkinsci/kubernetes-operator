setup() {
    load 'test_helper'
    _common_setup
}

diag() {
    # Use this function like that:
    #    diag "$output"
    # if you want to debug some test
    echo "# DEBUG $@" >&3
}

#bats test_tags=phase:setup
@test "1.0 Create namespace" {
    ${KUBECTL} get ns ${DETIK_CLIENT_NAMESPACE} && skip "Namespace ${DETIK_CLIENT_NAMESPACE} already exists"
    run ${KUBECTL} create ns ${DETIK_CLIENT_NAMESPACE}
    assert_success
}

#bats test_tags=phase:helm
@test "1.1 Vanilla install helm chart" {
    run echo ${DETIK_CLIENT_NAMESPACE}
    run echo ${OPERATOR_IMAGE}
    ${HELM} status default && skip "Helm release 'default' already exists"
    run ${HELM} install default \
      --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
      --set namespace=${DETIK_CLIENT_NAMESPACE} \
      --set operator.image=${OPERATOR_IMAGE} \
      chart/jenkins-operator
    assert_success
    assert ${HELM} status default
    touch "chart/jenkins-operator/deploy.tmp"
}

@test "1.2 Helm: check Jenkins operator pods status" {
  run verify "there is 1 deployment named 'default-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'default-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'default-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

@test "1.3 Helm: check Jenkins service status" {
  run verify "there is 1 service named 'jenkins-operator-http-jenkins'"
  assert_success

  run verify "there is 1 service named 'jenkins-operator-slave-jenkins'"
  assert_success
}

@test "1.4 Helm: check Jenkins Pod status" {
  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success

  run try "at most 20 times every 5s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

@test "1.5 Helm: check Jenkins configmaps created" {
  run verify "there is 1 configmap named 'jenkins-operator-base-configuration-jenkins'"
  assert_success
  run verify "there is 1 configmap named 'jenkins-operator-init-configuration-jenkins'"
  assert_success
  run verify "there is 1 configmap named 'jenkins-operator-scripts-jenkins'"
  assert_success
}

@test "1.6 Helm: check Jenkins operator role status" {
  run verify "there are 2 role named 'jenkins-operator*'"
  assert_success
  run verify "there is 1 role named 'leader-election-role'"
  assert_success
}

@test "1.7 Helm: check Jenkins operator role binding status" {
  run verify "there is 1 rolebinding named 'jenkins-operator-jenkins'"
  assert_success
  run verify "there is 1 rolebinding named 'leader-election-rolebinding'"
  assert_success
}

@test "1.8 Helm: check Jenkins operator service account status" {
  run verify "there are 2 serviceaccount named 'jenkins-operator*'"
  assert_success
}
