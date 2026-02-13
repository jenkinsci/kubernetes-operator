package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JenkinsSpec defines the desired state of Jenkins
// +k8s:openapi-gen=true
type JenkinsSpec struct {
	// Master represents Jenkins master pod properties and Jenkins plugins.
	// Every single change here requires a pod restart.
	Master JenkinsMaster `json:"master"`

	// SeedJobs defines list of Jenkins Seed Job configurations
	// More info: https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-seed-jobs-and-pipelines/
	// +optional
	SeedJobs []SeedJob `json:"seedJobs,omitempty"`

	// SeedJobAgentImage defines the image that will be used by the seed job agent. If not defined jenkins/inbound-agent:4.9-1 will be used.
	// +optional
	SeedJobAgentImage string `json:"seedJobAgentImage,omitempty"`

	// ValidateSecurityWarnings enables or disables validating potential security warnings in Jenkins plugins via admission webhooks.
	//+optional
	ValidateSecurityWarnings bool `json:"validateSecurityWarnings,omitempty"`

	// Notifications defines list of a services which are used to inform about Jenkins status
	// Can be used to integrate chat services like Slack, Microsoft Teams or Mailgun
	// +optional
	Notifications []Notification `json:"notifications,omitempty"`

	// Service is Kubernetes service of Jenkins master HTTP pod
	// Defaults to :
	// port: 8080
	// type: ClusterIP
	// +optional
	Service Service `json:"service,omitempty"`

	// Service is Kubernetes service of Jenkins slave pods
	// Defaults to :
	// port: 50000
	// type: ClusterIP
	// +optional
	SlaveService Service `json:"slaveService,omitempty"`

	// Backup defines configuration of Jenkins backup
	// More info: https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-backup-and-restore/
	// +optional
	Backup Backup `json:"backup,omitempty"`

	// Backup defines configuration of Jenkins backup restore
	// More info: https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-backup-and-restore/
	// +optional
	Restore Restore `json:"restore,omitempty"`

	// GroovyScripts defines configuration of Jenkins customization via groovy scripts
	// +optional
	GroovyScripts GroovyScripts `json:"groovyScripts,omitempty"`

	// ConfigurationAsCode defines configuration of Jenkins customization via Configuration as Code Jenkins plugin
	// +optional
	ConfigurationAsCode ConfigurationAsCode `json:"configurationAsCode,omitempty"`

	// Roles defines list of extra RBAC roles for the Jenkins Master pod service account
	// +optional
	Roles []rbacv1.RoleRef `json:"roles,omitempty"`

	// ServiceAccount defines Jenkins master service account attributes
	// +optional
	ServiceAccount ServiceAccount `json:"serviceAccount,omitempty"`

	// JenkinsAPISettings defines configuration used by the operator to gain admin access to the Jenkins API
	JenkinsAPISettings JenkinsAPISettings `json:"jenkinsAPISettings"`

	// +optional
	Lifecycle JenkinsLifecycle `json:"lifecycle,omitempty"`
}

type JenkinsLifecycle struct {
	// +optional
	Ignore JenkinsLifecycleIgnore `json:"ignore,omitempty"`
}

type JenkinsLifecycleIgnore struct {
	IgnoredVolumes     []string `json:"volumes,omitempty"`
	IgnoredEnvs        []string `json:"envs,omitempty"`
	IgnoredAnnotations []string `json:"annotations,omitempty"`
	IgnoredLabels      []string `json:"labels,omitempty"`
}

// AuthorizationStrategy defines authorization strategy of the operator for the Jenkins API
type AuthorizationStrategy string

const (
	// CreateUserAuthorizationStrategy operator sets HudsonPrivateSecurityRealm and FullControlOnceLoggedInAuthorizationStrategy than creates user using init.d groovy script
	CreateUserAuthorizationStrategy AuthorizationStrategy = "createUser"
	// ServiceAccountAuthorizationStrategy operator gets token associated with Jenkins service account and uses it as bearer token
	ServiceAccountAuthorizationStrategy AuthorizationStrategy = "serviceAccount"
)

// JenkinsAPISettings defines configuration used by the operator to gain admin access to the Jenkins API
type JenkinsAPISettings struct {
	AuthorizationStrategy AuthorizationStrategy `json:"authorizationStrategy"`
}

// ServiceAccount defines Kubernetes service account attributes
type ServiceAccount struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NotificationLevel defines the level of a Notification.
type NotificationLevel string

const (
	// NotificationLevelWarning - Only Warnings
	NotificationLevelWarning NotificationLevel = "warning"

	// NotificationLevelInfo - Only info
	NotificationLevelInfo NotificationLevel = "info"
)

// Notification is a service configuration used to send notifications about Jenkins status.
type Notification struct {
	LoggingLevel NotificationLevel `json:"level"`
	Verbose      bool              `json:"verbose"`
	Name         string            `json:"name"`
	Slack        *Slack            `json:"slack,omitempty"`
	Teams        *MicrosoftTeams   `json:"teams,omitempty"`
	Mailgun      *Mailgun          `json:"mailgun,omitempty"`
	SMTP         *SMTP             `json:"smtp,omitempty"`
}

// Slack is handler for Slack notification channel.
type Slack struct {
	// The web hook URL to Slack App
	WebHookURLSecretKeySelector SecretKeySelector `json:"webHookURLSecretKeySelector"`
}

// SMTP is handler for sending emails via this protocol.
type SMTP struct {
	UsernameSecretKeySelector SecretKeySelector `json:"usernameSecretKeySelector"`
	PasswordSecretKeySelector SecretKeySelector `json:"passwordSecretKeySelector"`
	Port                      int               `json:"port"`
	Server                    string            `json:"server"`
	TLSInsecureSkipVerify     bool              `json:"tlsInsecureSkipVerify,omitempty"`
	From                      string            `json:"from"`
	To                        string            `json:"to"`
}

// MicrosoftTeams is handler for Microsoft MicrosoftTeams notification channel.
type MicrosoftTeams struct {
	// The web hook URL to MicrosoftTeams App
	WebHookURLSecretKeySelector SecretKeySelector `json:"webHookURLSecretKeySelector"`
}

// Mailgun is handler for Mailgun email service notification channel.
type Mailgun struct {
	Domain                  string            `json:"domain"`
	APIKeySecretKeySelector SecretKeySelector `json:"apiKeySecretKeySelector"`
	Recipient               string            `json:"recipient"`
	From                    string            `json:"from"`
}

// SecretKeySelector selects a key of a Secret.
type SecretKeySelector struct {
	// The name of the secret in the pod's namespace to select from.
	corev1.LocalObjectReference `json:"secret"`
	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`
}

// Container defines Kubernetes container attributes.
type Container struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	Name string `json:"name"`

	// Docker image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	Image string `json:"image"`

	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`

	// Compute Resources required by this container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Resources corev1.ResourceRequirements `json:"resources"`

	// Entrypoint array. Not executed within a shell.
	// The docker image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty"`

	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty"`

	// Container's working directory.
	// If not specified, the container runtime's default will be used, which
	// might be configured in the container image.
	// +optional
	WorkingDir string `json:"workingDir,omitempty"`

	// List of ports to expose from the container. Exposing a port here gives
	// the system additional information about the network connections a
	// container uses, but is primarily informational. Not specifying a port here
	// DOES NOT prevent that port from being exposed. Any port which is
	// listening on the default "0.0.0.0" address inside a container will be
	// accessible from the network.
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// List of environment variables to set in the container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Pod volumes to mount into the container's filesystem.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Actions that the management system should take in response to container lifecycle events.
	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty"`

	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
}

// Plugin defines Jenkins plugin.
type Plugin struct {
	// Name is the name of Jenkins plugin
	Name string `json:"name"`
	// Version is the version of Jenkins plugin
	Version string `json:"version"`
	// DownloadURL is the custom url from where plugin has to be downloaded.
	DownloadURL string `json:"downloadURL,omitempty"`
}

// JenkinsMaster defines the Jenkins master pod attributes and plugins,
// every single change requires a Jenkins master pod restart.
type JenkinsMaster struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// SecurityContext that applies to all the containers of the Jenkins
	// Master. As per kubernetes specification, it can be overridden
	// for each container individually.
	// +optional
	// Defaults to:
	// runAsUser: 1000
	// fsGroup: 1000
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// List of containers belonging to the pod.
	// Containers cannot currently be added or removed.
	// There must be at least one container in a Pod.
	// +optional
	// Defaults to:
	// - image: jenkins/jenkins:lts
	//   imagePullPolicy: Always
	//   livenessProbe:
	//     failureThreshold: 12
	//     httpGet:
	//       path: /login
	//       port: http
	//       scheme: HTTP
	//     initialDelaySeconds: 80
	//     periodSeconds: 10
	//     successThreshold: 1
	//     timeoutSeconds: 5
	//   lifecycle: {}
	//   name: jenkins-master
	//   readinessProbe:
	//     failureThreshold: 3
	//     httpGet:
	//       path: /login
	//       port: http
	//       scheme: HTTP
	//     initialDelaySeconds: 30
	//     periodSeconds: 10
	//     successThreshold: 1
	//     timeoutSeconds: 1
	//   resources:
	//     limits:
	//       cpu: 1500m
	//       memory: 3Gi
	//     requests:
	//       cpu: "1"
	//       memory: 600Mi
	Containers []Container `json:"containers,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// List of volumes that can be mounted by containers belonging to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// BasePlugins contains plugins required by operator
	// +optional
	// Defaults to :
	// - name: configuration-as-code
	// version: "1625.v27444588cc3d"
	// - name: git
	// version: "5.0.0"
	// - name: job-dsl
	// version: "1.83"
	// - name: kubernetes
	// version: "3909.v1f2c633e8590"
	// - name: kubernetes-credentials-provider
	// version: "1.211.vc236a_f5a_2f3c"
	// - name: workflow-aggregator
	// version: "596.v8c21c963d92d"
	// - name: workflow-job
	// version: "1289.vd1c337fd5354"
	BasePlugins []Plugin `json:"basePlugins,omitempty"`

	// Plugins contains plugins required by user
	// +optional
	Plugins []Plugin `json:"plugins,omitempty"`

	// Allow to override jenkins-plugin-cli default behavior
	// while downloading the plugin and dependencies
	// see: https://github.com/jenkinsci/plugin-installation-manager-tool#cli-options
	// +optional
	LatestPlugins *bool `json:"latestPlugins,omitempty"`

	// Allow to skip installation of both BasePlugins and Plugins.
	// Requires using a custom image which includes the BasePlugins.
	// Defaults to false.
	// +optional
	SkipPlugins *bool `json:"skipPlugins,omitempty"`

	// DisableCSRFProtection allows you to toggle CSRF Protection on Jenkins
	DisableCSRFProtection bool `json:"disableCSRFProtection"`

	// PriorityClassName for Jenkins master pod
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// HostAliases for Jenkins master pod and SeedJob agent
	// +optional
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`

	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// Defaults to 30 seconds.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
}

// Service defines Kubernetes service attributes
type Service struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Route service traffic to pods with label keys and values matching this
	// selector. If empty or not present, the service is assumed to have an
	// external process managing its endpoints, which Kubernetes will not
	// modify. Only applies to types ClusterIP, NodePort, and LoadBalancer.
	// Ignored if type is ExternalName.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Type determines how the Service is exposed. Defaults to ClusterIP. Valid
	// options are ExternalName, ClusterIP, NodePort, and LoadBalancer.
	// "ExternalName" maps to the specified externalName.
	// "ClusterIP" allocates a cluster-internal IP address for load-balancing to
	// endpoints. Endpoints are determined by the selector or if that is not
	// specified, by manual construction of an Endpoints object. If clusterIP is
	// "None", no virtual IP is allocated and the endpoints are published as a
	// set of endpoints rather than a stable IP.
	// "NodePort" builds on ClusterIP and allocates a port on every node which
	// routes to the clusterIP.
	// "LoadBalancer" builds on NodePort and creates an
	// external load-balancer (if supported in the current cloud) which routes
	// to the clusterIP.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services---service-types
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// The port that are exposed by this service.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	Port int32 `json:"port,omitempty"`

	// The port on each node on which this service is exposed when type=NodePort or LoadBalancer.
	// Usually assigned by the system. If specified, it will be allocated to the service
	// if unused or else creation of the service will fail.
	// Default is to auto-allocate a port if the ServiceType of this Service requires one.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`

	// If specified and supported by the platform, this will restrict traffic through the cloud-provider
	// load-balancer will be restricted to the specified client IPs. This field will be ignored if the
	// cloud-provider does not support the feature."
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/#restricting-cloud-metadata-api-access
	// +optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`

	// Only applies to Service Type: LoadBalancer
	// LoadBalancer will get created with the IP specified in this field.
	// This feature depends on whether the underlying cloud-provider supports specifying
	// the loadBalancerIP when a load balancer is created.
	// This field will be ignored if the cloud-provider does not support the feature.
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
type JenkinsStatus struct {
	// OperatorVersion is the operator version which manages this CR
	// +optional
	OperatorVersion string `json:"operatorVersion,omitempty"`

	// ProvisionStartTime is a time when Jenkins master pod has been created
	// +optional
	ProvisionStartTime *metav1.Time `json:"provisionStartTime,omitempty"`

	// BaseConfigurationCompletedTime is a time when Jenkins base configuration phase has been completed
	// +optional
	BaseConfigurationCompletedTime *metav1.Time `json:"baseConfigurationCompletedTime,omitempty"`

	// UserConfigurationCompletedTime is a time when Jenkins user configuration phase has been completed
	// +optional
	UserConfigurationCompletedTime *metav1.Time `json:"userConfigurationCompletedTime,omitempty"`

	// RestoredBackup is the restored backup number after Jenkins master pod restart
	// +optional
	RestoredBackup uint64 `json:"restoredBackup,omitempty"`

	// LastBackup is the latest backup number
	// +optional
	LastBackup uint64 `json:"lastBackup,omitempty"`

	// PendingBackup is the pending backup number
	// +optional
	PendingBackup uint64 `json:"pendingBackup,omitempty"`

	// BackupDoneBeforePodDeletion tells if backup before pod deletion has been made
	// +optional
	BackupDoneBeforePodDeletion bool `json:"backupDoneBeforePodDeletion,omitempty"`

	// UserAndPasswordHash is a SHA256 hash made from user and password
	// +optional
	UserAndPasswordHash string `json:"userAndPasswordHash,omitempty"`

	// CreatedSeedJobs contains list of seed job id already created in Jenkins
	// +optional
	CreatedSeedJobs []string `json:"createdSeedJobs,omitempty"`

	// AppliedGroovyScripts is a list with all applied groovy scripts in Jenkins by the operator
	// +optional
	AppliedGroovyScripts []AppliedGroovyScript `json:"appliedGroovyScripts,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jenkins is the Schema for the jenkins API
// +k8s:openapi-gen=true
type Jenkins struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Jenkins
	Spec JenkinsSpec `json:"spec,omitempty"`

	// Status defines the observed state of Jenkins
	Status JenkinsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsList contains a list of Jenkins
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}

// JenkinsCredentialType defines type of Jenkins credential used to seed job mechanism.
type JenkinsCredentialType string

const (
	// NoJenkinsCredentialCredentialType define none Jenkins credential type
	NoJenkinsCredentialCredentialType JenkinsCredentialType = ""
	// BasicSSHCredentialType define basic SSH Jenkins credential type
	BasicSSHCredentialType JenkinsCredentialType = "basicSSHUserPrivateKey"
	// UsernamePasswordCredentialType define username & password Jenkins credential type
	UsernamePasswordCredentialType JenkinsCredentialType = "usernamePassword"
	GithubAppCredentialType        JenkinsCredentialType = "githubApp"
	// ExternalCredentialType defines other credential type
	ExternalCredentialType JenkinsCredentialType = "external"
)

// AllowedJenkinsCredentialMap contains all allowed Jenkins credentials types.
var AllowedJenkinsCredentialMap = map[string]string{
	string(NoJenkinsCredentialCredentialType): "",
	string(BasicSSHCredentialType):            "",
	string(UsernamePasswordCredentialType):    "",
	string(GithubAppCredentialType):           "",
	string(ExternalCredentialType):            "",
}

// SeedJob defines configuration for seed job
// More info: https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuration/#configure-seed-jobs-and-pipelines.
type SeedJob struct {
	// ID is the unique seed job name
	ID string `json:"id,omitempty"`

	// CredentialID is the Kubernetes secret name which stores repository access credentials
	CredentialID string `json:"credentialID,omitempty"`

	// Description is the description of the seed job
	// +optional
	Description string `json:"description,omitempty"`

	// Targets is the repository path where are seed job definitions
	Targets string `json:"targets,omitempty"`

	// RepositoryBranch is the repository branch where are seed job definitions
	RepositoryBranch string `json:"repositoryBranch,omitempty"`

	// RepositoryURL is the repository access URL. Can be SSH or HTTPS.
	RepositoryURL string `json:"repositoryUrl,omitempty"`

	// JenkinsCredentialType is the https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/ credential type
	// +optional
	JenkinsCredentialType JenkinsCredentialType `json:"credentialType,omitempty"`

	// BitbucketPushTrigger is used for Bitbucket web hooks
	// +optional
	BitbucketPushTrigger bool `json:"bitbucketPushTrigger"`

	// GitHubPushTrigger is used for GitHub web hooks
	// +optional
	GitHubPushTrigger bool `json:"githubPushTrigger"`

	// BuildPeriodically is setting for scheduled trigger
	// +optional
	BuildPeriodically string `json:"buildPeriodically"`

	// PollSCM is setting for polling changes in SCM
	// +optional
	PollSCM string `json:"pollSCM"`

	// IgnoreMissingFiles is setting for Job DSL API plugin to ignore files that miss
	// +optional
	IgnoreMissingFiles bool `json:"ignoreMissingFiles"`

	// AdditionalClasspath is setting for Job DSL API plugin to set Additional Classpath
	// +optional
	AdditionalClasspath string `json:"additionalClasspath"`

	// FailOnMissingPlugin is setting for Job DSL API plugin that fails job if required plugin is missing
	// +optional
	FailOnMissingPlugin bool `json:"failOnMissingPlugin"`

	// UnstableOnDeprecation is setting for Job DSL API plugin that sets build status as unstable if build using deprecated features
	// +optional
	UnstableOnDeprecation bool `json:"unstableOnDeprecation"`
}

// Handler defines a specific action that should be taken.
type Handler struct {
	// Exec specifies the action to take.
	Exec *corev1.ExecAction `json:"exec,omitempty"`
}

// Backup defines configuration of Jenkins backup.
type Backup struct {
	// ContainerName is the container name responsible for backup operation
	ContainerName string `json:"containerName"`

	// Action defines action which performs backup in backup container sidecar
	Action Handler `json:"action"`

	// Interval tells how often make backup in seconds
	// Defaults to 30.
	Interval uint64 `json:"interval"`

	// MakeBackupBeforePodDeletion tells operator to make backup before Jenkins master pod deletion
	MakeBackupBeforePodDeletion bool `json:"makeBackupBeforePodDeletion"`
}

// Restore defines configuration of Jenkins backup restore operation.
type Restore struct {
	// ContainerName is the container name responsible for restore backup operation
	ContainerName string `json:"containerName"`

	// Action defines action which performs restore backup in restore container sidecar
	Action Handler `json:"action"`

	// GetLatestAction defines action which returns the latest backup number. If there is no backup "-1" should be
	// returned.
	// +optional
	GetLatestAction Handler `json:"getLatestAction"`

	// RecoveryOnce if want to restore specific backup set this field and then Jenkins will be restarted and desired backup will be restored
	// +optional
	RecoveryOnce uint64 `json:"recoveryOnce,omitempty"`
}

// AppliedGroovyScript is the applied groovy script in Jenkins by the operator.
type AppliedGroovyScript struct {
	// ConfigurationType is the name of the configuration type(base-groovy, user-groovy, user-casc)
	ConfigurationType string `json:"configurationType"`
	// Source is the name of source where is located groovy script
	Source string `json:"source"`
	// Name is the name of the groovy script
	Name string `json:"name"`
	// Hash is the hash of the groovy script and secrets which it uses
	Hash string `json:"hash"`
}

// SecretRef is reference to Kubernetes secret.
type SecretRef struct {
	Name string `json:"name"`
}

// ConfigMapRef is reference to Kubernetes ConfigMap.
type ConfigMapRef struct {
	Name string `json:"name"`
}

// Customization defines configuration of Jenkins customization.
type Customization struct {
	Secret         SecretRef      `json:"secret"`
	Configurations []ConfigMapRef `json:"configurations"`
}

// GroovyScripts defines configuration of Jenkins customization via groovy scripts.
type GroovyScripts struct {
	Customization `json:",inline"`
}

// ConfigurationAsCode defines configuration of Jenkins customization via Configuration as Code Jenkins plugin.
type ConfigurationAsCode struct {
	Customization `json:",inline"`
}
