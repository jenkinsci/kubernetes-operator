package plugins

const (
	configurationAsCodePlugin           = "configuration-as-code:1625.v27444588cc3d"
	gitPlugin                           = "git:5.0.2"
	jobDslPlugin                        = "job-dsl:1.83"
	kubernetesPlugin                    = "kubernetes:3937.vd7b_82db_e347b_"
	kubernetesCredentialsProviderPlugin = "kubernetes-credentials-provider:1.211.vc236a_f5a_2f3c"
	workflowAggregatorPlugin            = "workflow-aggregator:596.v8c21c963d92d"
	workflowJobPlugin                   = "workflow-job:1301.v054d9cea_9593"
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
