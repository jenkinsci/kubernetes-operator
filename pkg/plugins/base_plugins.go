package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1569.vb_72405b_80249"
	gitPlugin                           = "git:5.0.0"
	jobDslPlugin                        = "job-dsl:1.82"
	kubernetesPlugin                    = "kubernetes:3896.v19b_160fd9589"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.209.v862c6e5fb_1ef"
	workflowAggregatorPlugin            = "workflow-aggregator:596.v8c21c963d92d"
	workflowJobPlugin                   = "workflow-job:1284.v2fe8ed4573d4"
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
