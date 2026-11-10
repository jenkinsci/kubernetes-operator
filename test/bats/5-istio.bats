setup() {
  load 'test_helper'
  _common_setup
  load 'mesh_helper'
}

#bats test_tags=phase:setup,scenario:istio
@test "5.0 Install istio" {
  run make mesh-install MESH=istio
  assert_success
}

#bats test_tags=phase:helm,scenario:istio
@test "5.1 Reconcile Jenkins with istio regular sidecar" {
  mesh_scenario istio regular
}

#bats test_tags=phase:helm,scenario:istio
@test "5.2 Reconcile Jenkins with istio native sidecar" {
  mesh_scenario istio native
}

#bats test_tags=phase:helm,scenario:istio
@test "5.3 Clean istio" {
  mesh_cleanup istio
}
