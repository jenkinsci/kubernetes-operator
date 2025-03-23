_common_setup() {
    export BATS_LIB_PATH="${BATS_LIB_PATH}:/usr/lib"
    bats_load_library bats-support
    bats_load_library bats-assert
    bats_load_library bats-file
    bats_load_library bats-detik/detik.bash

    CONTEXT="kind-jenkins"
    export DETIK_CLIENT_NAME="kubectl"
    export DETIK_CLIENT_NAMESPACE="ns-$(git rev-parse --verify HEAD --short)"
    export KUBECTL="kubectl --context=${CONTEXT} -n ${DETIK_CLIENT_NAMESPACE}"
    export HELM="helm --kube-context=${CONTEXT} -n ${DETIK_CLIENT_NAMESPACE}"
}

get_latest_chart_version() {
    helm search repo jenkins-operator/jenkins --versions | awk 'NR==2 {print $2}' | sed 's/v//'
}
