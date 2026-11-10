package resources

import (
	"encoding/json"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBaseConfigurationUsageStatistics(t *testing.T) {
	for _, tt := range []struct {
		name, master string
		disabled     bool
	}{
		{name: "omitted", master: `{}`},
		{name: "false", master: `{"disableUsageStatistics":false}`},
		{name: "true", master: `{"disableUsageStatistics":true}`, disabled: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			j := &v1alpha2.Jenkins{}
			require.NoError(t, json.Unmarshal([]byte(`{"spec":{"master":`+tt.master+`}}`), j))
			j.Spec.Master.Containers = []v1alpha2.Container{{Name: "jenkins-master"}}
			cm, err := NewBaseConfigurationConfigMap(metav1.ObjectMeta{}, j, "cluster.local")
			require.NoError(t, err)
			script, present := cm.Data[disableUsageStatsGroovyScriptName]
			require.Equal(t, tt.disabled, present)
			if tt.disabled {
				require.Contains(t, script, "jenkins.setNoUsageStatistics(true)")
			}
			require.Contains(t, cm.Data, enableCSRFGroovyScriptName)
		})
	}
}
