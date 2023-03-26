setup() {
  load 'test_helper'
  _common_setup
}

#bats test_tags=phase:helm
@test "2.1 Install helm chart with options" {
  run ${HELM} dependency update chart/jenkins-operator
  assert_success
  ${HELM} status options && skip "Helm release 'options' already exists"
  run ${HELM} install options \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set webhook.enabled=true \
    --set cert-manager.installCRDs=true \
    --set cert-manager.startupapicheck.enabled=true \
    chart/jenkins-operator
  assert_success
  assert ${HELM} status options
  touch "chart/jenkins-operator/deploy.tmp"
}

#bats test_tags=phase:helm
@test "2.2 Helm: check Jenkins operator pods status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run verify "there is 1 deployment named 'options-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'options-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'options-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm
@test "2.3 Helm: check Jenkins Pod status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success

  run try "at most 20 times every 5s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

@test "2.9 Helm: Clean" {
  run ${HELM} uninstall options
  assert_success
  rm "chart/jenkins-operator/deploy.tmp"
}
