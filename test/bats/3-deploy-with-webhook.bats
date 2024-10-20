setup() {
  load 'test_helper'
  _common_setup
}

#bats test_tags=phase:setup,scenario:webhook
@test "3.0  Init: create namespace" {
  ${KUBECTL} get ns ${DETIK_CLIENT_NAMESPACE} && skip "Namespace ${DETIK_CLIENT_NAMESPACE} already exists"
  run ${KUBECTL} create ns ${DETIK_CLIENT_NAMESPACE}
  assert_success
}

#bats test_tags=phase:setup,scenario:webhook
@test "3.1  Init: add helm chart repo" {
    ${HELM} repo list|grep -qc jenkins-operator && skip "Jenkins repo already exists"
    upstream_url="https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart"
    run ${HELM} repo add jenkins-operator $upstream_url
    assert_success
    assert_output '"jenkins-operator" has been added to your repositories'
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.2  Helm: install helm chart with webhook enabled" {
  run ${HELM} dependency update chart/jenkins-operator
  assert_success
  ${HELM} status webhook && skip "Helm release 'webhook' already exists"
  run ${HELM} install webhook \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.image="jenkins/jenkins:2.462.3-lts" \
    --set jenkins.imagePullPolicy="IfNotPresent" \
    --set jenkins.backup.makeBackupBeforePodDeletion=true \
    --set jenkins.backup.image=quay.io/jenkins-kubernetes-operator/backup-pvc:e2e-test \
    --set webhook.enabled=true \
    --set cert-manager.enabled=true \
    --set cert-manager.startupapicheck.enabled=true \
    jenkins-operator/jenkins-operator --version=$(get_latest_chart_version)
  assert_success
  assert ${HELM} status webhook
  touch "chart/jenkins-operator/deploy.tmp"
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.3  Helm: check Jenkins operator pods status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run verify "there is 1 deployment named 'webhook-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'webhook-jenkins-operator-'"
  assert_success

  run try "at most 50 times every 5s to get pod named 'webhook-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success

  run ${KUBECTL} rollout restart deployment webhook-jenkins-operator
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.4  Helm: check Jenkins Pod status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run try "at most 30 times every 10s to get pod named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.5  Helm: check Jenkins crd" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 crd named 'jenkins.jenkins.io'"
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.6  Helm: check cert-manager crd" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 crd named 'certificates.cert-manager.io'"
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.7  Helm: upgrade from main branch same value" {
  run ${HELM} dependency update chart/jenkins-operator
  assert_success
  run ${HELM} upgrade webhook \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.image="jenkins/jenkins:2.462.3-lts" \
    --set jenkins.imagePullPolicy="IfNotPresent" \
    --set jenkins.backup.makeBackupBeforePodDeletion=true \
    --set jenkins.backup.image=quay.io/jenkins-kubernetes-operator/backup-pvc:e2e-test \
    --set webhook.enabled=true \
    --set cert-manager.enabled=true \
    --set cert-manager.startupapicheck.enabled=true \
    chart/jenkins-operator --wait
  assert_success
  assert ${HELM} status webhook
  sleep 10
  touch "chart/jenkins-operator/deploy.tmp"
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.8  Helm: check Jenkins operator pods status again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run verify "there is 1 deployment named 'webhook-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'webhook-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 5s to get pods named 'webhook-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.9  Helm: check Jenkins Pod status again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.10 Helm: check Jenkins crd again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 crd named 'jenkins.jenkins.io'"
  assert_success
}

#bats test_tags=phase:helm,scenario:webhook
@test "3.11 Helm: check cert-manager crd again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 crd named 'certificates.cert-manager.io'"
  assert_success
}

@test "3.12 Helm: clean" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"

  run ${HELM} uninstall webhook --wait
  assert_success
  sleep 20

  run verify "there is 0 pvc named 'jenkins backup'"
  assert_success

  rm "chart/jenkins-operator/deploy.tmp"
}

#TODO: add another test scenario while installing from kubectl apply
