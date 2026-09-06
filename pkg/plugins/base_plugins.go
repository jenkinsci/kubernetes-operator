package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:2121.v86fe99d4b_b_a_b_"
	gitPlugin                           = "git:5.10.1"
	jobDslPlugin                        = "job-dsl:3732.v9a_c49a_61a_313"
	kubernetesPlugin                    = "kubernetes:4547.v52f3080db_8cd"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.315.v92008589c044"
	// Depends on workflow-job which should be automatically downloaded
	// Hardcoding the workflow-job version leads to frequent breakage
	workflowAggregatorPlugin = "workflow-aggregator:608.v67378e9d3db_1"
)

// basePluginsList contains plugins to install by operator.
var basePluginsList = []Plugin{
	Must(New(configurationAsCodePlugin)),
	Must(New(gitPlugin)),
	Must(New(jobDslPlugin)),
	Must(New(kubernetesPlugin)),
	Must(New(kubernetesCredentialsProviderPlugin)),
	Must(New(workflowAggregatorPlugin)),
}

// BasePlugins returns list of plugins to install by operator.
func BasePlugins() []Plugin {
	return basePluginsList
}
