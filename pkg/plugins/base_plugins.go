package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1932.v75cb_b_f1b_698d"
	gitPlugin                           = "git:5.7.0"
	jobDslPlugin                        = "job-dsl:1.89"
	kubernetesPlugin                    = "kubernetes:4295.v7fa_01b_309c95"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.262.v2670ef7ea_0c5"
	workflowAggregatorPlugin            = "workflow-aggregator:600.vb_57cdd26fdd7"
	workflowJobPlugin                   = "workflow-job:1436.vfa_244484591f"
)

// basePluginsList contains plugins to install by operator.
var basePluginsList = []Plugin{
	Must(New(configurationAsCodePlugin)),
	Must(New(gitPlugin)),
	Must(New(jobDslPlugin)),
	Must(New(kubernetesPlugin)),
	Must(New(kubernetesCredentialsProviderPlugin)),
	Must(New(workflowAggregatorPlugin)),
	Must(New(workflowJobPlugin)),
}

// BasePlugins returns list of plugins to install by operator.
func BasePlugins() []Plugin {
	return basePluginsList
}
