package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1700.v6f448841296e"
	gitPlugin                           = "git:5.2.0"
	jobDslPlugin                        = "job-dsl:1.85"
	kubernetesPlugin                    = "kubernetes:4029.v5712230ccb_f8"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.234.vf3013b_35f5b_a"
	workflowAggregatorPlugin            = "workflow-aggregator:596.v8c21c963d92d"
	workflowJobPlugin                   = "workflow-job:1342.v046651d5b_dfe"
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
