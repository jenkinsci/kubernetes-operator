setup() {
  load 'test_helper'
  _common_setup
  load 'mesh_helper'
}

#bats test_tags=phase:setup,scenario:linkerd
@test "6.0 Install linkerd" {
  run make mesh-install MESH=linkerd
  assert_success
}

#bats test_tags=phase:helm,scenario:linkerd
@test "6.1 Reconcile Jenkins with linkerd regular sidecar" {
  mesh_scenario linkerd regular
}

#bats test_tags=phase:helm,scenario:linkerd
@test "6.2 Reconcile Jenkins with linkerd native sidecar" {
  mesh_scenario linkerd native
}

#bats test_tags=phase:helm,scenario:linkerd
@test "6.3 Clean linkerd" {
  mesh_cleanup linkerd
}
