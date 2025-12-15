package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:2006.v001a_2ca_6b_574"
	gitPlugin                           = "git:5.8.1"
	jobDslPlugin                        = "job-dsl:1.93"
	kubernetesPlugin                    = "kubernetes:4398.vb_b_33d9e7fe23"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.299.v610fa_e76761a_"
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
