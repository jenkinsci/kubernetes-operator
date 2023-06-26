setup() {
  load 'test_helper'
  _common_setup
}

#bats test_tags=phase:helm
@test "2.1 Install helm chart with options" {
  run ${HELM} dependency update chart/jenkins-operator
  assert_success
  run ${KUBECTL} label node jenkins-control-plane batstest=yep
  ${HELM} status options && skip "Helm release 'options' already exists"
  run ${HELM} install options \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.nodeSelector.batstest=yep \
    --set jenkins.backup.makeBackupBeforePodDeletion=false \
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

@test "2.4 check node selector" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  NODENAME=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath={.spec.nodeName})

  run ${KUBECTL} get node -l batstest=yep -o name
  assert_success
  assert_output "node/$NODENAME"
}

@test "2.5 check jenkins-plugin-cli command" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run ${KUBECTL} logs -c jenkins-master jenkins-jenkins
  assert_success
  assert_output --partial 'jenkins-plugin-cli --verbose --latest true -f /var/lib/jenkins/base-plugins.txt'
  assert_output --partial 'jenkins-plugin-cli --verbose --latest true -f /var/lib/jenkins/user-plugins.txt'
}


@test "2.7 check backup" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  sleep 120
  run ${KUBECTL} logs -l app.kubernetes.io/name=jenkins-operator --tail 10000
  assert_success
  assert_output --partial "Performing backup '1'"
  assert_output --partial "Backup completed '1', updating status"
}


@test "2. Helm: Clean" {
  skip
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run ${HELM} uninstall options
  assert_success

  rm "chart/jenkins-operator/deploy.tmp"
}
