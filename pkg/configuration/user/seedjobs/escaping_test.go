package seedjobs

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/stretchr/testify/require"
)

func TestSeedJobStringsAreLiteral(t *testing.T) {
	for _, input := range []struct{ name, value, literal string }{
		{"breakout", `"); System.setProperty('operator.seedjobs.injected', 'yes'); println("`, `'"); System.setProperty(\'operator.seedjobs.injected\', \'yes\'); println("'`},
		{"interpolation", `${System.setProperty('operator.seedjobs.injected', 'yes')}`, `'${System.setProperty(\'operator.seedjobs.injected\', \'yes\')}'`},
		{"backslashes and newlines", "path\\folder\nnext\tline", `'path\\folder\nnext\tline'`},
	} {
		payload, literal := input.value, input.literal
		for _, tt := range []struct {
			name string
			set  func(*v1alpha2.SeedJob)
			want []string
		}{
			{"id", func(j *v1alpha2.SeedJob) { j.ID = payload }, []string{"def jobDslSeedName = " + literal + " + '-' + 'job-dsl-seed'", "jobRef.setDisplayName('Seed Job from ' + " + literal + ")"}},
			{"repositoryUrl", func(j *v1alpha2.SeedJob) { j.RepositoryURL = payload }, []string{"GitSCM.createRepoList(" + literal + ", ''"}},
			{"credentialID", func(j *v1alpha2.SeedJob) { j.CredentialID = payload }, []string{"GitSCM.createRepoList('https://example.com/repo.git', " + literal + ")"}},
			{"repositoryBranch", func(j *v1alpha2.SeedJob) { j.RepositoryBranch = payload }, []string{"new BranchSpec(" + literal + ")"}},
			{"targets", func(j *v1alpha2.SeedJob) { j.Targets = payload }, []string{"executeDslScripts.setTargets(" + literal + ")"}},
			{"additionalClasspath", func(j *v1alpha2.SeedJob) { j.AdditionalClasspath = payload }, []string{"executeDslScripts.setAdditionalClasspath(" + literal + ")"}},
			{"pollSCM", func(j *v1alpha2.SeedJob) { j.PollSCM = payload }, []string{"new SCMTrigger(" + literal + ")"}},
			{"buildPeriodically", func(j *v1alpha2.SeedJob) { j.BuildPeriodically = payload }, []string{"new TimerTrigger(" + literal + ")"}},
		} {
			t.Run(tt.name+"/"+input.name, func(t *testing.T) {
				j := v1alpha2.SeedJob{ID: "example", RepositoryURL: "https://example.com/repo.git", RepositoryBranch: "main", Targets: "jobs/*.groovy"}
				tt.set(&j)
				script, err := seedJobCreatingGroovyScript(j)
				require.NoError(t, err)
				for _, want := range tt.want {
					require.Contains(t, script, want)
				}
			})
		}
	}
}

func TestSeedJobRenderingCompatibility(t *testing.T) {
	j := v1alpha2.SeedJob{ID: "example", RepositoryURL: "https://example.com/repo.git", RepositoryBranch: "main", Targets: "jobs/*.groovy"}
	script, err := seedJobCreatingGroovyScript(j)
	require.NoError(t, err)
	require.NotContains(t, script, "new SCMTrigger(")
	require.NotContains(t, script, "new TimerTrigger(")
	require.NotContains(t, script, "new GitHubPushTrigger(")
	require.NotContains(t, script, "new BitBucketTrigger(")
	j.PollSCM = "H/5 * * * *"
	j.BuildPeriodically = "H 1 * * *\nH 2 * * *"
	j.GitHubPushTrigger = true
	j.BitbucketPushTrigger = true
	j.IgnoreMissingFiles = true
	j.FailOnMissingPlugin = true
	j.UnstableOnDeprecation = true
	script, err = seedJobCreatingGroovyScript(j)
	require.NoError(t, err)
	for _, want := range []string{
		"def jobDslSeedName = 'example' + '-' + 'job-dsl-seed'",
		"jobRef.setDisplayName('Seed Job from ' + 'example')",
		"GitSCM.createRepoList('https://example.com/repo.git', '')",
		"new BranchSpec('main')", "setTargets('jobs/*.groovy')", "setAdditionalClasspath('')",
		"new SCMTrigger('H/5 * * * *')", `new TimerTrigger('H 1 * * *\nH 2 * * *')`,
		"new GitHubPushTrigger()", "new BitBucketTrigger()", "new LabelAtom('seed-job-agent')",
		"setIgnoreMissingFiles(true)", "setFailOnMissingPlugin(true)", "setUnstableOnDeprecation(true)",
	} {
		require.Contains(t, script, want)
	}
	again, err := seedJobCreatingGroovyScript(j)
	require.NoError(t, err)
	require.Equal(t, script, again)
}
