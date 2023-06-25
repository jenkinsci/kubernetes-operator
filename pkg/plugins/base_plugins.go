package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1647.ve39ca_b_829b_42"
	gitPlugin                           = "git:5.1.0"
	jobDslPlugin                        = "job-dsl:1.84"
	kubernetesPlugin                    = "kubernetes:3952.v88e3b_0cf300b_"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.225.v14f9e6b_28f53"
	workflowAggregatorPlugin            = "workflow-aggregator:596.v8c21c963d92d"
	workflowJobPlugin                   = "workflow-job:1308.v58d48a_763b_31"
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
