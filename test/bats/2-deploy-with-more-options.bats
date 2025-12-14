setup() {
  load 'test_helper'
  _common_setup
}

#bats test_tags=phase:setup,scenario:more-options
@test "2.0  Init: create namespace" {
  ${KUBECTL} get ns ${DETIK_CLIENT_NAMESPACE} && skip "Namespace ${DETIK_CLIENT_NAMESPACE} already exists"
  run ${KUBECTL} create ns ${DETIK_CLIENT_NAMESPACE}
  assert_success
}

#bats test_tags=phase:setup,scenario:more-options
@test "2.1  Init: add helm chart repo" {
    ${HELM} repo list|grep -qc jenkins-operator && skip "Jenkins repo already exists"
    upstream_url="https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart"
    run ${HELM} repo add jenkins-operator $upstream_url
    assert_success
    assert_output '"jenkins-operator" has been added to your repositories'
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.2  Helm: install helm chart with options" {
  run ${KUBECTL} label node jenkins-control-plane batstest=yep
  assert_success
  run ${HELM} install options \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.nodeSelector.batstest=yep \
    --set jenkins.image="jenkins/jenkins:2.528.3-lts" \
    --set jenkins.imagePullPolicy="IfNotPresent" \
    --set jenkins.backup.makeBackupBeforePodDeletion=false \
    --set jenkins.backup.image=quay.io/jenkins-kubernetes-operator/backup-pvc:e2e-test \
    jenkins-operator/jenkins-operator --version=$(get_latest_chart_version)
  assert_success
  assert ${HELM} status options
  touch "chart/jenkins-operator/deploy.tmp"
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.3  Helm: check Jenkins operator pods status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run verify "there is 1 deployment named 'options-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'options-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'options-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.4  Helm: check Jenkins Pod status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run try "at most 20 times every 5s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"backup\")].ready' is 'true'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.5  Helm: check node selector" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  NODENAME=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath={.spec.nodeName})

  run ${KUBECTL} get node -l batstest=yep -o name
  assert_success
  assert_output "node/$NODENAME"
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.6  Helm: check jenkins-plugin-cli command" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run ${KUBECTL} logs -c jenkins-master jenkins-jenkins
  assert_success
  assert_output --partial 'jenkins-plugin-cli --verbose --latest true -f /var/lib/jenkins/base-plugins.txt'
  assert_output --partial 'jenkins-plugin-cli --verbose --latest true -f /var/lib/jenkins/user-plugins.txt'
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.8  Helm: check backup" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  sleep 120

  # use --tail -1 to get all logs and reduce flakiness
  # using -l to select a label usually sets --tail 10
  run ${KUBECTL} logs -l app.kubernetes.io/name=jenkins-operator --tail -1

  assert_success
  assert_output --partial "Performing backup '1'"
  assert_output --partial "Backup completed '1', updating status"
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.9  Helm: upgrade from main branch same value" {
  run ${HELM} upgrade options \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.nodeSelector.batstest=yep \
    --set jenkins.image="jenkins/jenkins:2.528.3-lts" \
    --set jenkins.imagePullPolicy="IfNotPresent" \
    --set jenkins.lifecycle.preStop.exec.command="{echo bats-test}" \
    --set jenkins.backup.makeBackupBeforePodDeletion=false \
    --set jenkins.backup.image=quay.io/jenkins-kubernetes-operator/backup-pvc:e2e-test \
    chart/jenkins-operator --wait
  assert_success
  assert ${HELM} status options
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.10 Helm: check Jenkins operator pods status again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run verify "there is 1 deployment named 'options-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'options-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'options-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.11 Helm: check Jenkins Pod status again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success

  run try "at most 20 times every 5s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.12  Helm: check lifecycle hook injection" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.spec.containers[?(@.name==\"jenkins-master\")].lifecycle.preStop.exec.command[0]' is 'echo bats-test'"
  assert_success
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.13 Helm: check node selector again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  NODENAME=$(${KUBECTL} get pod jenkins-jenkins -o jsonpath={.spec.nodeName})

  run ${KUBECTL} get node -l batstest=yep -o name
  assert_success
  assert_output "node/$NODENAME"
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.14 Helm: check jenkins-plugin-cli command again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  # Check logs for jenkins-plugin-cli command with retry.
  # Retry is necessary here to reduce flakiness due to additional delays
  # from reconciling and recreating jenkins pods after helm upgrade.
  # We assert success only after the retry loop to reduce noise.
  LOG_CMD="${KUBECTL} logs -c jenkins-master jenkins-jenkins"

  EXPECTED_LOG_LINE_BASE_PLUGINS="jenkins-plugin-cli --verbose --latest true -f /var/lib/jenkins/base-plugins.txt"
  retry 10 10 "${LOG_CMD} | grep -e '${EXPECTED_LOG_LINE_BASE_PLUGINS}'"

  run $LOG_CMD
  assert_success
  assert_output --partial "${EXPECTED_LOG_LINE_BASE_PLUGINS}"

  EXPECTED_LOG_LINE_USER_PLUGINS="jenkins-plugin-cli --verbose --latest true -f /var/lib/jenkins/user-plugins.txt"
  retry 10 10 "${LOG_CMD} | grep -e '${EXPECTED_LOG_LINE_USER_PLUGINS}'"

  run $LOG_CMD
  assert_success
  assert_output --partial "${EXPECTED_LOG_LINE_USER_PLUGINS}"
}

#bats test_tags=phase:helm,scenario:more-options
@test "2.15 Helm: clean" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run ${HELM} uninstall options --wait
  assert_success
  sleep 10

  run verify "there is 0 pvc named 'jenkins backup'"
  assert_success

  rm "chart/jenkins-operator/deploy.tmp"
}
