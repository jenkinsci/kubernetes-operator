//go:build !OpenShift && !OpenShiftOAuth
// +build !OpenShift,!OpenShiftOAuth

package e2e

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
)

const (
// skipTestSafeRestart   = false
// skipTestPriorityClass = false
)

func updateJenkinsCR(jenkins *v1alpha2.Jenkins) {
	// do nothing
}
