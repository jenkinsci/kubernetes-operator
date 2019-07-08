package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsSpec defines the desired state of Jenkins
// +k8s:openapi-gen=true
type JenkinsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Master       JenkinsMaster `json:"master,omitempty"`
	SeedJobs     []SeedJob     `json:"seedJobs,omitempty"`
	Service      Service       `json:"service,omitempty"`
	SlaveService Service       `json:"slaveService,omitempty"`
}

// Container defines Kubernetes container attributes
type Container struct {
	Name            string                      `json:"name"`
	Image           string                      `json:"image"`
	Command         []string                    `json:"command,omitempty"`
	Args            []string                    `json:"args,omitempty"`
	WorkingDir      string                      `json:"workingDir,omitempty"`
	Ports           []corev1.ContainerPort      `json:"ports,omitempty"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Env             []corev1.EnvVar             `json:"env,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	VolumeMounts    []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	LivenessProbe   *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe  *corev1.Probe               `json:"readinessProbe,omitempty"`
	Lifecycle       *corev1.Lifecycle           `json:"lifecycle,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	SecurityContext *corev1.SecurityContext     `json:"securityContext,omitempty"`
}

// JenkinsMaster defines the Jenkins master pod attributes and plugins,
// every single change requires Jenkins master pod restart
type JenkinsMaster struct {
	Container

	// pod properties
	Annotations  map[string]string `json:"masterAnnotations,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Containers   []Container       `json:"containers,omitempty"`
	Volumes      []corev1.Volume   `json:"volumes,omitempty"`

	// OperatorPlugins contains plugins required by operator
	OperatorPlugins map[string][]string `json:"basePlugins,omitempty"`
	// Plugins contains plugins required by user
	Plugins map[string][]string `json:"plugins,omitempty"`
}

// Service defines Kubernetes service attributes which Operator will manage
type Service struct {
	Annotations              map[string]string  `json:"annotations,omitempty"`
	Labels                   map[string]string  `json:"labels,omitempty"`
	Type                     corev1.ServiceType `json:"type,omitempty"`
	Port                     int32              `json:"port,omitempty"`
	NodePort                 int32              `json:"nodePort,omitempty"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
type JenkinsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	OperatorVersion                string       `json:"operatorVersion,omitempty"`
	ProvisionStartTime             *metav1.Time `json:"provisionStartTime,omitempty"`
	BaseConfigurationCompletedTime *metav1.Time `json:"baseConfigurationCompletedTime,omitempty"`
	UserConfigurationCompletedTime *metav1.Time `json:"userConfigurationCompletedTime,omitempty"`
	Builds                         []Build      `json:"builds,omitempty"`
}

// BuildStatus defines type of Jenkins build job status
type BuildStatus string

const (
	// BuildSuccessStatus - the build had no errors
	BuildSuccessStatus BuildStatus = "success"
	// BuildUnstableStatus - the build had some errors but they were not fatal. For example, some tests failed
	BuildUnstableStatus BuildStatus = "unstable"
	// BuildNotBuildStatus - this status code is used in a multi-stage build (like maven2) where a problem in earlier stage prevented later stages from building
	BuildNotBuildStatus BuildStatus = "not_build"
	// BuildFailureStatus - the build had a fatal error
	BuildFailureStatus BuildStatus = "failure"
	// BuildAbortedStatus - the build was manually aborted
	BuildAbortedStatus BuildStatus = "aborted"
	// BuildRunningStatus - this is custom build status for running build, not present in jenkins build result
	BuildRunningStatus BuildStatus = "running"
	// BuildExpiredStatus - this is custom build status for expired build, not present in jenkins build result
	BuildExpiredStatus BuildStatus = "expired"
)

// Build defines Jenkins Build status with corresponding metadata
type Build struct {
	JobName        string       `json:"jobName,omitempty"`
	Hash           string       `json:"hash,omitempty"`
	Number         int64        `json:"number,omitempty"`
	Status         BuildStatus  `json:"status,omitempty"`
	Retires        int          `json:"retries,omitempty"`
	CreateTime     *metav1.Time `json:"createTime,omitempty"`
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jenkins is the Schema for the jenkins API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Jenkins struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsSpec   `json:"spec,omitempty"`
	Status JenkinsStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsList contains a list of Jenkins
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}

// JenkinsCredentialType defines type of Jenkins credential used to seed job mechanisms
type JenkinsCredentialType string

const (
	// NoJenkinsCredentialCredentialType define none Jenkins credential type
	NoJenkinsCredentialCredentialType JenkinsCredentialType = ""
	// BasicSSHCredentialType define basic SSH Jenkins credential type
	BasicSSHCredentialType JenkinsCredentialType = "basicSSHUserPrivateKey"
	// UsernamePasswordCredentialType define username & password Jenkins credential type
	UsernamePasswordCredentialType JenkinsCredentialType = "usernamePassword"
)

// AllowedJenkinsCredentialMap contains all allowed Jenkins credentials types
var AllowedJenkinsCredentialMap = map[string]string{
	string(NoJenkinsCredentialCredentialType): "",
	string(BasicSSHCredentialType):            "",
	string(UsernamePasswordCredentialType):    "",
}

// SeedJob defined configuration for seed jobs and deploy keys
type SeedJob struct {
	ID                    string                `json:"id,omitempty"`
	CredentialID          string                `json:"credentialID,omitempty"`
	Description           string                `json:"description,omitempty"`
	Targets               string                `json:"targets,omitempty"`
	RepositoryBranch      string                `json:"repositoryBranch,omitempty"`
	RepositoryURL         string                `json:"repositoryUrl,omitempty"`
	JenkinsCredentialType JenkinsCredentialType `json:"credentialType,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Jenkins{}, &JenkinsList{})
}
