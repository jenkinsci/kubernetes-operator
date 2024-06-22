package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1810.v9b_c30a_249a_4c"
	gitPlugin                           = "git:5.2.2"
	jobDslPlugin                        = "job-dsl:1.87"
	kubernetesPlugin                    = "kubernetes:4246.v5a_12b_1fe120e"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.262.v2670ef7ea_0c5"
	workflowAggregatorPlugin            = "workflow-aggregator:596.v8c21c963d92d"
	workflowJobPlugin                   = "workflow-job:1400.v7fd111b_ec82f"
)

// basePluginsList contains plugins to install by operator.
var basePluginsList = []Plugin{
	Must(New(configurationAsCodePlugin)),
	Must(New(gitPlugin)),
	Must(New(jobDslPlugin)),
	Must(New(kubernetesPlugin)),
	Must(New(kubernetesCredentialsProviderPlugin)),
	Must(New(workflowJobPlugin)),
	Must(New(workflowAggregatorPlugin)),
}

// BasePlugins returns list of plugins to install by operator.
func BasePlugins() []Plugin {
	return basePluginsList
}
