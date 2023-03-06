package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1569.vb_72405b_80249"
	gitPlugin                           = "git:5.0.0"
	jobDslPlugin                        = "job-dsl:1.81"
	kubernetesPlugin                    = "kubernetes:3883.v4d70a_a_a_df034"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.209.v862c6e5fb_1ef"
	workflowAggregatorPlugin            = "workflow-aggregator:590.v6a_d052e5a_a_b_5"
	workflowJobPlugin                   = "workflow-job:1282.ve6d865025906"
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
