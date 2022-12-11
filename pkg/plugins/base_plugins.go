package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1569.vb_72405b_80249"
	gitPlugin                           = "git:4.14.3"
	jobDslPlugin                        = "job-dsl:1.81"
	kubernetesPlugin                    = "kubernetes:3734.v562b_b_a_627ea_c"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.206.v7ce2cf7b_0c8b"
	workflowAggregatorPlugin            = "workflow-aggregator:590.v6a_d052e5a_a_b_5"
	workflowJobPlugin                   = "workflow-job:1254.v3f64639b_11dd"
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
