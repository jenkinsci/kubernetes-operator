package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1775.v810dc950b_514"
	gitPlugin                           = "git:5.2.1"
	jobDslPlugin                        = "job-dsl:1.87"
	kubernetesPlugin                    = "kubernetes:4186.v1d804571d5d4"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.258.v95949f923a_a_e"
	workflowAggregatorPlugin            = "workflow-aggregator:596.v8c21c963d92d"
	workflowJobPlugin                   = "workflow-job:1385.vb_58b_86ea_fff1"
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
