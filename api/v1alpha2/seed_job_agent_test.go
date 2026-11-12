package v1alpha2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedAgentRestrictionSerialization(t *testing.T) {
	for _, input := range []string{`{}`, `{"seedJobRestrictJobsToLabel":true}`, `{"seedJobRestrictJobsToLabel":false}`} {
		var spec JenkinsSpec
		require.NoError(t, json.Unmarshal([]byte(input), &spec))
		data, err := json.Marshal(spec)
		require.NoError(t, err)
		var copy JenkinsSpec
		require.NoError(t, json.Unmarshal(data, &copy))
		require.Equal(t, spec.SeedJobRestrictJobsToLabel, copy.SeedJobRestrictJobsToLabel)
		if spec.SeedJobRestrictJobsToLabel != nil {
			cloned := (&Jenkins{Spec: spec}).DeepCopy()
			*cloned.Spec.SeedJobRestrictJobsToLabel = !*spec.SeedJobRestrictJobsToLabel
			require.NotEqual(t, *spec.SeedJobRestrictJobsToLabel, *cloned.Spec.SeedJobRestrictJobsToLabel)
		}
	}
}
