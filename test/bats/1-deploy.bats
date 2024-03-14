setup() {
  load 'test_helper'
  _common_setup
}

diag() {
  echo "# DEBUG $@" >&3
}

#bats test_tags=phase:setup,scenario:vanilla
@test "1.0  Init: create namespace" {
  ${KUBECTL} get ns ${DETIK_CLIENT_NAMESPACE} && skip "Namespace ${DETIK_CLIENT_NAMESPACE} already exists"
  run ${KUBECTL} create ns ${DETIK_CLIENT_NAMESPACE}
  assert_success
}

#bats test_tags=phase:setup,scenario:vanilla
@test "1.1  Init: add helm chart repo" {
    ${HELM} repo list|grep -qc jenkins-operator && skip "Jenkins repo already exists"
    upstream_url="https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart"
    run ${HELM} repo add jenkins-operator $upstream_url
    assert_success
    assert_output '"jenkins-operator" has been added to your repositories'
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.2  Helm: vanilla install helm chart latest tagged version" {
  run echo ${DETIK_CLIENT_NAMESPACE}
  run echo ${OPERATOR_IMAGE}
  ${HELM} status default && skip "Helm release 'default' already exists"
  run ${HELM} install default \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.image="jenkins/jenkins:2.440.1-lts" \
    --set jenkins.backup.makeBackupBeforePodDeletion=false \
    jenkins-operator/jenkins-operator --version=$(cat ../../VERSION.txt | sed 's/v//')
  assert_success
  assert ${HELM} status default
  touch "chart/jenkins-operator/deploy.tmp"
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.2  Helm: check Jenkins operator pods status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 deployment named 'default-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'default-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'default-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.3  Helm: check Jenkins Pod status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success

  run try "at most 20 times every 5s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.4  Helm: check Jenkins service status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 service named 'jenkins-operator-http-jenkins'"
  assert_success

  run verify "there is 1 service named 'jenkins-operator-slave-jenkins'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.5  Helm: check Jenkins configmaps created" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 configmap named 'jenkins-operator-base-configuration-jenkins'"
  assert_success
  run verify "there is 1 configmap named 'jenkins-operator-init-configuration-jenkins'"
  assert_success
  run verify "there is 1 configmap named 'jenkins-operator-scripts-jenkins'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.6  Helm: check Jenkins operator role status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there are 2 role named 'jenkins-operator*'"
  assert_success
  run verify "there is 1 role named 'leader-election-role'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.7  Helm: check Jenkins operator role binding status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 rolebinding named 'jenkins-operator-jenkins'"
  assert_success
  run verify "there is 1 rolebinding named 'leader-election-rolebinding'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.8  Helm: check Jenkins operator service account status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there are 2 serviceaccount named 'jenkins-operator*'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.9  Helm: check Jenkins crd" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 crd named 'jenkins.jenkins.io'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.10 Helm: upgrade from main branch same values" {
  run echo ${DETIK_CLIENT_NAMESPACE}
  run echo ${OPERATOR_IMAGE}
  run ${HELM} upgrade default \
    --set jenkins.namespace=${DETIK_CLIENT_NAMESPACE} \
    --set namespace=${DETIK_CLIENT_NAMESPACE} \
    --set operator.image=${OPERATOR_IMAGE} \
    --set jenkins.latestPlugins=true \
    --set jenkins.image="jenkins/jenkins:2.440.1-lts" \
    --set jenkins.backup.makeBackupBeforePodDeletion=false \
    chart/jenkins-operator
  assert_success
  assert ${HELM} status default
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.11 Helm: check Jenkins operator pods status again" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 deployment named 'default-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'default-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'default-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.12 Helm: check Jenkins operator pods status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run verify "there is 1 deployment named 'default-jenkins-operator'"
  assert_success

  run verify "there is 1 pod named 'default-jenkins-operator-'"
  assert_success

  run try "at most 20 times every 10s to get pods named 'default-jenkins-operator-' and verify that '.status.containerStatuses[?(@.name==\"jenkins-operator\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.13 Helm: check Jenkins Pod status" {
  [[ ! -f "chart/jenkins-operator/deploy.tmp" ]] && skip "Jenkins helm chart have not been deployed correctly"
  run try "at most 20 times every 10s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success

  run try "at most 20 times every 5s to get pods named 'jenkins-jenkins' and verify that '.status.containerStatuses[?(@.name==\"jenkins-master\")].ready' is 'true'"
  assert_success
}

#bats test_tags=phase:helm,scenario:vanilla
@test "1.14 Helm: clean" {
  run ${HELM} uninstall default
  assert_success
  # Wait for the complete removal
  sleep 30

  run verify "there is 0 pvc named 'jenkins backup'"
  assert_success

  rm "chart/jenkins-operator/deploy.tmp"
}
