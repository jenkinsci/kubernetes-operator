// +build !OpenShift
// +build !OpenShiftOAuth

package e2e

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
)

const (
	skipTestSafeRestart   = false
	skipTestPriorityClass = false
)

func updateJenkinsCR(t *testing.T, jenkins *v1alpha2.Jenkins) {
	t.Log("Update Jenkins CR")
	// do nothing
}
