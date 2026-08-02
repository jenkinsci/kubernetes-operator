package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:2108.v02b_430db_0cf5"
	gitPlugin                           = "git:5.10.1"
	jobDslPlugin                        = "job-dsl:3654.vdf58f53e2d15"
	kubernetesPlugin                    = "kubernetes:4540.v612369217f87"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.303.vdfcf47fb_b_fef"
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
