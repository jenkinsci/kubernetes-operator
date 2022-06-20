package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1346.ve8cfa_3473c94"
	gitPlugin                           = "git:4.11.3"
	jobDslPlugin                        = "job-dsl:1.78.1"
	kubernetesPlugin                    = "kubernetes:1.31.3"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:0.20"
	workflowAggregatorPlugin            = "workflow-aggregator:2.6"
	workflowJobPlugin                   = "workflow-job:1145.v7f2433caa07f"
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
