package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	basicSettingsGroovyScriptName               = "1-basic-settings.groovy"
	enableCSRFGroovyScriptName                  = "2-enable-csrf.groovy"
	disableUsageStatsGroovyScriptName           = "3-disable-usage-stats.groovy"
	disableInsecureFeaturesGroovyScriptName     = "4-disable-insecure-features.groovy"
	configureKubernetesPluginGroovyScriptName   = "5-configure-kubernetes-plugin.groovy"
	configureViewsGroovyScriptName              = "6-configure-views.groovy"
	disableJobDslScriptApprovalGroovyScriptName = "7-disable-job-dsl-script-approval.groovy"
)

const basicSettingsFmt = `
import jenkins.model.Jenkins
import jenkins.model.JenkinsLocationConfiguration
import hudson.model.Node.Mode

def jenkins = Jenkins.instance
//Number of jobs that run simultaneously on master.
jenkins.setNumExecutors(%d)
//Jobs must specify that they want to run on master
jenkins.setMode(Mode.EXCLUSIVE)
jenkins.save()
`

const enableCSRF = `
import hudson.security.csrf.DefaultCrumbIssuer
import jenkins.model.Jenkins

def jenkins = Jenkins.instance

if (jenkins.getCrumbIssuer() == null) {
    jenkins.setCrumbIssuer(new DefaultCrumbIssuer(true))
    jenkins.save()
    println('CSRF Protection enabled.')
} else {
    println('CSRF Protection already configured.')
}
`

const disableUsageStats = `
import jenkins.model.Jenkins

def jenkins = Jenkins.instance

if (jenkins.isUsageStatisticsCollected()) {
    jenkins.setNoUsageStatistics(true)
    jenkins.save()
    println('Jenkins usage stats submitting disabled.')
} else {
    println('Nothing changed.  Usage stats are not submitted to the Jenkins project.')
}
`

const disableInsecureFeatures = `
import jenkins.*
import jenkins.model.*
import hudson.model.*
import jenkins.security.s2m.*

def jenkins = Jenkins.instance

println("Disabling insecure Jenkins features...")

println("Disabling insecure protocols...")
println("Old protocols: [" + jenkins.getAgentProtocols().join(", ") + "]")
HashSet<String> newProtocols = new HashSet<>(jenkins.getAgentProtocols())
newProtocols.removeAll(Arrays.asList("JNLP3-connect", "JNLP2-connect", "JNLP-connect", "CLI-connect"))
println("New protocols: [" + newProtocols.join(", ") + "]")
jenkins.setAgentProtocols(newProtocols)

println("Disabling CLI access of /cli URL...")
def remove = { list ->
    list.each { item ->
        if (item.getClass().name.contains("CLIAction")) {
            println("Removing extension ${item.getClass().name}")
            list.remove(item)
        }
    }
}
remove(jenkins.getExtensionList(RootAction.class))
remove(jenkins.actions)

if (jenkins.getDescriptor("jenkins.CLI") != null) {
    jenkins.getDescriptor("jenkins.CLI").get().setEnabled(false)
}

jenkins.save()
`

const configureKubernetesPluginFmt = `
import com.cloudbees.plugins.credentials.CredentialsScope
import com.cloudbees.plugins.credentials.SystemCredentialsProvider
import com.cloudbees.plugins.credentials.domains.Domain
import jenkins.model.Jenkins
import org.csanchez.jenkins.plugins.kubernetes.KubernetesCloud

def jenkins = Jenkins.getInstance()

def kubernetes = Jenkins.instance.clouds.getByName("kubernetes")
def add = false
if (kubernetes == null) {
    add = true
	kubernetes = new KubernetesCloud("kubernetes")
}
kubernetes.setServerUrl("https://kubernetes.default.svc.%s:443")
kubernetes.setNamespace("%s")
kubernetes.setJenkinsUrl("%s")
kubernetes.setJenkinsTunnel("%s")
kubernetes.setRetentionTimeout(15)
if (add) {
	jenkins.clouds.add(kubernetes)
}

jenkins.save()
`

const configureViews = `
import hudson.model.ListView
import jenkins.model.Jenkins

def Jenkins jenkins = Jenkins.getInstance()

def seedViewName = 'seed-jobs'
def nonSeedViewName = 'non-seed-jobs'

if (jenkins.getView(seedViewName) == null) {
    def seedView = new ListView(seedViewName)
    seedView.setIncludeRegex('.*` + constants.SeedJobSuffix + `.*')
    jenkins.addView(seedView)
}

if (jenkins.getView(nonSeedViewName) == null) {
    def nonSeedView = new ListView(nonSeedViewName)
    nonSeedView.setIncludeRegex('((?!seed)(?!jenkins).)*')
    jenkins.addView(nonSeedView)
}

jenkins.save()
`

const disableJobDSLScriptApproval = `
import jenkins.model.Jenkins
import javaposse.jobdsl.plugin.GlobalJobDslSecurityConfiguration
import jenkins.model.GlobalConfiguration

// disable Job DSL script approval
GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).useScriptSecurity=false
GlobalConfiguration.all().get(GlobalJobDslSecurityConfiguration.class).save()
`

// GetBaseConfigurationConfigMapName returns name of Kubernetes config map used to base configuration.
func GetBaseConfigurationConfigMapName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-base-configuration-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// NewBaseConfigurationConfigMap builds Kubernetes config map used to base configuration.
func NewBaseConfigurationConfigMap(meta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins, kubernetesClusterDomain string) (*corev1.ConfigMap, error) {
	meta.Name = GetBaseConfigurationConfigMapName(jenkins)
	clusterDomain, err := getClusterDomain(kubernetesClusterDomain)
	if err != nil {
		return nil, err
	}
	jenkinsServiceFQDN, err := GetJenkinsHTTPServiceFQDN(jenkins, kubernetesClusterDomain)
	if err != nil {
		return nil, err
	}
	jenkinsSlavesServiceFQDN, err := GetJenkinsSlavesServiceFQDN(jenkins, kubernetesClusterDomain)
	if err != nil {
		return nil, err
	}
	suffix := ""
	if prefix, ok := GetJenkinsOpts(*jenkins)["prefix"]; ok {
		suffix = prefix
	}
	groovyScriptsMap := map[string]string{
		basicSettingsGroovyScriptName:           fmt.Sprintf(basicSettingsFmt, constants.DefaultAmountOfExecutors),
		enableCSRFGroovyScriptName:              enableCSRF,
		disableUsageStatsGroovyScriptName:       disableUsageStats,
		disableInsecureFeaturesGroovyScriptName: disableInsecureFeatures,
		configureKubernetesPluginGroovyScriptName: fmt.Sprintf(configureKubernetesPluginFmt,
			clusterDomain,
			jenkins.ObjectMeta.Namespace,
			fmt.Sprintf("http://%s:%d%s", jenkinsServiceFQDN, jenkins.Spec.Service.Port, suffix),
			fmt.Sprintf("%s:%d", jenkinsSlavesServiceFQDN, jenkins.Spec.SlaveService.Port),
		),
		configureViewsGroovyScriptName:              configureViews,
		disableJobDslScriptApprovalGroovyScriptName: disableJobDSLScriptApproval,
	}

	if jenkins.Spec.Master.DisableCSRFProtection {
		delete(groovyScriptsMap, enableCSRFGroovyScriptName)
	}
	return &corev1.ConfigMap{
		TypeMeta:   buildConfigMapTypeMeta(),
		ObjectMeta: meta,
		Data:       groovyScriptsMap,
	}, nil
}
