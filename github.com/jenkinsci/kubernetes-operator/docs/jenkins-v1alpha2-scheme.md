<p>Packages:</p>
<ul>
<li>
<a href="#jenkins.io">jenkins.io</a>
</li>
</ul>
<h2 id="jenkins.io">jenkins.io</h2>
<p>
<p>Package v1alpha2 contains API Schema definitions for the jenkins.io v1alpha2 API group</p>
</p>
Resource Types:
<ul><li>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.Jenkins">Jenkins</a>
</li></ul>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Jenkins">Jenkins
</h3>
<p>
<p>Jenkins is the Schema for the jenkins API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
jenkins.io/v1alpha2
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Jenkins</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsSpec">
JenkinsSpec
</a>
</em>
</td>
<td>
<p>Spec defines the desired state of the Jenkins</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>master</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsMaster">
JenkinsMaster
</a>
</em>
</td>
<td>
<p>Master represents Jenkins master pod properties and Jenkins plugins.
Every single change here requires a pod restart.</p>
</td>
</tr>
<tr>
<td>
<code>seedJobs</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.SeedJob">
[][]github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.SeedJob
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SeedJobs defines list of Jenkins Seed Job configurations
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-seed-jobs-and-pipelines">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-seed-jobs-and-pipelines</a></p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service">
Service
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service is Kubernetes service of Jenkins master HTTP pod
Defaults to :
port: 8080
type: ClusterIP</p>
</td>
</tr>
<tr>
<td>
<code>slaveService</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service">
Service
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service is Kubernetes service of Jenkins slave pods
Defaults to :
port: 50000
type: ClusterIP</p>
</td>
</tr>
<tr>
<td>
<code>backup</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Backup">
Backup
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Backup defines configuration of Jenkins backup
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore</a></p>
</td>
</tr>
<tr>
<td>
<code>restore</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Restore">
Restore
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Backup defines configuration of Jenkins backup restore
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore</a></p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsStatus">
JenkinsStatus
</a>
</em>
</td>
<td>
<p>Status defines the observed state of Jenkins</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Backup">Backup
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsSpec">JenkinsSpec</a>)
</p>
<p>
<p>Backup defines configuration of Jenkins backup</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>containerName</code></br>
<em>
string
</em>
</td>
<td>
<p>ContainerName is the container name responsible for backup operation</p>
</td>
</tr>
<tr>
<td>
<code>action</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Handler">
Handler
</a>
</em>
</td>
<td>
<p>Action defines action which performs backup in backup container sidecar</p>
</td>
</tr>
<tr>
<td>
<code>interval</code></br>
<em>
uint64
</em>
</td>
<td>
<p>Interval tells how often make backup in seconds
Defaults to 30.</p>
</td>
</tr>
<tr>
<td>
<code>makeBackupBeforePodDeletion</code></br>
<em>
bool
</em>
</td>
<td>
<p>MakeBackupBeforePodDeletion tells operator to make backup before Jenkins master pod deletion</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Build">Build
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsStatus">JenkinsStatus</a>)
</p>
<p>
<p>Build defines Jenkins Build status with corresponding metadata</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>jobName</code></br>
<em>
string
</em>
</td>
<td>
<p>JobName is the Jenkins job name</p>
</td>
</tr>
<tr>
<td>
<code>hash</code></br>
<em>
string
</em>
</td>
<td>
<p>Hash is the unique data identifier used in build</p>
</td>
</tr>
<tr>
<td>
<code>number</code></br>
<em>
int64
</em>
</td>
<td>
<p>Number is the Jenkins build number</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.BuildStatus">
BuildStatus
</a>
</em>
</td>
<td>
<p>Status is the status of Jenkins build</p>
</td>
</tr>
<tr>
<td>
<code>retries</code></br>
<em>
int
</em>
</td>
<td>
<p>Retires is the amount of Jenkins job build retries</p>
</td>
</tr>
<tr>
<td>
<code>createTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>CreateTime is the time when the first build has been created</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastUpdateTime is the last update status time</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.BuildStatus">BuildStatus
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.Build">Build</a>)
</p>
<p>
<p>BuildStatus defines type of Jenkins build job status</p>
</p>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Container">Container
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsMaster">JenkinsMaster</a>)
</p>
<p>
<p>Container defines Kubernetes container attributes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the container specified as a DNS_LABEL.
Each container in a pod must have a unique name (DNS_LABEL).</p>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>Docker image name.
More info: <a href="https://kubernetes.io/docs/concepts/containers/images">https://kubernetes.io/docs/concepts/containers/images</a></p>
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#pullpolicy-v1-core">
Kubernetes core/v1.PullPolicy
</a>
</em>
</td>
<td>
<p>Image pull policy.
One of Always, Never, IfNotPresent.
Defaults to Always.</p>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>Compute Resources required by this container.
More info: <a href="https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/">https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</a></p>
</td>
</tr>
<tr>
<td>
<code>command</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Entrypoint array. Not executed within a shell.
The docker image&rsquo;s ENTRYPOINT is used if this is not provided.
Variable references $(VAR_NAME) are expanded using the container&rsquo;s environment. If a variable
cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
regardless of whether the variable exists or not.
More info: <a href="https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell">https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</a></p>
</td>
</tr>
<tr>
<td>
<code>args</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Arguments to the entrypoint.
The docker image&rsquo;s CMD is used if this is not provided.
Variable references $(VAR_NAME) are expanded using the container&rsquo;s environment. If a variable
cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
regardless of whether the variable exists or not.
More info: <a href="https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell">https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</a></p>
</td>
</tr>
<tr>
<td>
<code>workingDir</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Container&rsquo;s working directory.
If not specified, the container runtime&rsquo;s default will be used, which
might be configured in the container image.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#containerport-v1-core">
[]Kubernetes core/v1.ContainerPort
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of ports to expose from the container. Exposing a port here gives
the system additional information about the network connections a
container uses, but is primarily informational. Not specifying a port here
DOES NOT prevent that port from being exposed. Any port which is
listening on the default &ldquo;0.0.0.0&rdquo; address inside a container will be
accessible from the network.</p>
</td>
</tr>
<tr>
<td>
<code>envFrom</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#envfromsource-v1-core">
[]Kubernetes core/v1.EnvFromSource
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of sources to populate environment variables in the container.
The keys defined within a source must be a C_IDENTIFIER. All invalid keys
will be reported as an event when the container is starting. When a key exists in multiple
sources, the value associated with the last source will take precedence.
Values defined by an Env with a duplicate key will take precedence.</p>
</td>
</tr>
<tr>
<td>
<code>env</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#envvar-v1-core">
[]Kubernetes core/v1.EnvVar
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of environment variables to set in the container.</p>
</td>
</tr>
<tr>
<td>
<code>volumeMounts</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#volumemount-v1-core">
[]Kubernetes core/v1.VolumeMount
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Pod volumes to mount into the container&rsquo;s filesystem.</p>
</td>
</tr>
<tr>
<td>
<code>livenessProbe</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#probe-v1-core">
Kubernetes core/v1.Probe
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Periodic probe of container liveness.
Container will be restarted if the probe fails.</p>
</td>
</tr>
<tr>
<td>
<code>readinessProbe</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#probe-v1-core">
Kubernetes core/v1.Probe
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Periodic probe of container service readiness.
Container will be removed from service endpoints if the probe fails.</p>
</td>
</tr>
<tr>
<td>
<code>lifecycle</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#lifecycle-v1-core">
Kubernetes core/v1.Lifecycle
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Actions that the management system should take in response to container lifecycle events.</p>
</td>
</tr>
<tr>
<td>
<code>securityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#securitycontext-v1-core">
Kubernetes core/v1.SecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Security options the pod should run with.
More info: <a href="https://kubernetes.io/docs/concepts/policy/security-context/">https://kubernetes.io/docs/concepts/policy/security-context/</a>
More info: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/security-context/">https://kubernetes.io/docs/tasks/configure-pod-container/security-context/</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Handler">Handler
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.Backup">Backup</a>, 
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.Restore">Restore</a>)
</p>
<p>
<p>Handler defines a specific action that should be taken</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>exec</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#execaction-v1-core">
Kubernetes core/v1.ExecAction
</a>
</em>
</td>
<td>
<p>Exec specifies the action to take.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsCredentialType">JenkinsCredentialType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.SeedJob">SeedJob</a>)
</p>
<p>
<p>JenkinsCredentialType defines type of Jenkins credential used to seed job mechanism</p>
</p>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsMaster">JenkinsMaster
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsSpec">JenkinsSpec</a>)
</p>
<p>
<p>JenkinsMaster defines the Jenkins master pod attributes and plugins,
every single change requires a Jenkins master pod restart</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>masterAnnotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations is an unstructured key value map stored with a resource that may be
set by external tools to store and retrieve arbitrary metadata. They are not
queryable and should be preserved when modifying objects.
More info: <a href="http://kubernetes.io/docs/user-guide/annotations">http://kubernetes.io/docs/user-guide/annotations</a></p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NodeSelector is a selector which must be true for the pod to fit on a node.
Selector which must match a node&rsquo;s labels for the pod to be scheduled on that node.
More info: <a href="https://kubernetes.io/docs/concepts/configuration/assign-pod-node/">https://kubernetes.io/docs/concepts/configuration/assign-pod-node/</a></p>
</td>
</tr>
<tr>
<td>
<code>securityContext</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#podsecuritycontext-v1-core">
Kubernetes core/v1.PodSecurityContext
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecurityContext that applies to all the containers of the Jenkins
Master. As per kubernetes specification, it can be overridden
for each container individually.
Defaults to:
runAsUser: 1000
fsGroup: 1000</p>
</td>
</tr>
<tr>
<td>
<code>containers</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Container">
[][]github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Container
</a>
</em>
</td>
<td>
<p>List of containers belonging to the pod.
Containers cannot currently be added or removed.
There must be at least one container in a Pod.
Defaults to:
- image: jenkins/jenkins:lts
imagePullPolicy: Always
livenessProbe:
failureThreshold: 12
httpGet:
path: /login
port: http
scheme: HTTP
initialDelaySeconds: 80
periodSeconds: 10
successThreshold: 1
timeoutSeconds: 5
name: jenkins-master
readinessProbe:
failureThreshold: 3
httpGet:
path: /login
port: http
scheme: HTTP
initialDelaySeconds: 30
periodSeconds: 10
successThreshold: 1
timeoutSeconds: 1
resources:
limits:
cpu: 1500m
memory: 3Gi
requests:
cpu: &ldquo;1&rdquo;
memory: 600Mi</p>
</td>
</tr>
<tr>
<td>
<code>volumes</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#volume-v1-core">
[]Kubernetes core/v1.Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>List of volumes that can be mounted by containers belonging to the pod.
More info: <a href="https://kubernetes.io/docs/concepts/storage/volumes">https://kubernetes.io/docs/concepts/storage/volumes</a></p>
</td>
</tr>
<tr>
<td>
<code>basePlugins</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Plugin">
[][]github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Plugin
</a>
</em>
</td>
<td>
<p>BasePlugins contains plugins required by operator
Defaults to :
- name: kubernetes
version: 1.15.7
- name: workflow-job
version: &ldquo;2.32&rdquo;
- name: workflow-aggregator
version: &ldquo;2.6&rdquo;
- name: git
version: 3.10.0
- name: job-dsl
version: &ldquo;1.74&rdquo;
- name: configuration-as-code
version: &ldquo;1.19&rdquo;
- name: configuration-as-code-support
version: &ldquo;1.19&rdquo;
- name: kubernetes-credentials-provider
version: 0.12.1</p>
</td>
</tr>
<tr>
<td>
<code>plugins</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Plugin">
[][]github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Plugin
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Plugins contains plugins required by user</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsSpec">JenkinsSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.Jenkins">Jenkins</a>)
</p>
<p>
<p>JenkinsSpec defines the desired state of the Jenkins</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>master</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsMaster">
JenkinsMaster
</a>
</em>
</td>
<td>
<p>Master represents Jenkins master pod properties and Jenkins plugins.
Every single change here requires a pod restart.</p>
</td>
</tr>
<tr>
<td>
<code>seedJobs</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.SeedJob">
[][]github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.SeedJob
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SeedJobs defines list of Jenkins Seed Job configurations
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-seed-jobs-and-pipelines">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-seed-jobs-and-pipelines</a></p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service">
Service
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service is Kubernetes service of Jenkins master HTTP pod
Defaults to :
port: 8080
type: ClusterIP</p>
</td>
</tr>
<tr>
<td>
<code>slaveService</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service">
Service
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Service is Kubernetes service of Jenkins slave pods
Defaults to :
port: 50000
type: ClusterIP</p>
</td>
</tr>
<tr>
<td>
<code>backup</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Backup">
Backup
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Backup defines configuration of Jenkins backup
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore</a></p>
</td>
</tr>
<tr>
<td>
<code>restore</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Restore">
Restore
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Backup defines configuration of Jenkins backup restore
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-backup-and-restore</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsStatus">JenkinsStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.Jenkins">Jenkins</a>)
</p>
<p>
<p>JenkinsStatus defines the observed state of Jenkins</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>operatorVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>OperatorVersion is the operator version which manages this CR</p>
</td>
</tr>
<tr>
<td>
<code>provisionStartTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProvisionStartTime is a time when Jenkins master pod has been created</p>
</td>
</tr>
<tr>
<td>
<code>baseConfigurationCompletedTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BaseConfigurationCompletedTime is a time when Jenkins base configuration phase has been completed</p>
</td>
</tr>
<tr>
<td>
<code>userConfigurationCompletedTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>UserConfigurationCompletedTime is a time when Jenkins user configuration phase has been completed</p>
</td>
</tr>
<tr>
<td>
<code>builds</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Build">
[][]github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Build
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Builds contains Jenkins builds statues</p>
</td>
</tr>
<tr>
<td>
<code>restoredBackup</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>RestoredBackup is the restored backup number after Jenkins master pod restart</p>
</td>
</tr>
<tr>
<td>
<code>lastBackup</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastBackup is the latest backup number</p>
</td>
</tr>
<tr>
<td>
<code>pendingBackup</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>PendingBackup is the pending backup number</p>
</td>
</tr>
<tr>
<td>
<code>backupDoneBeforePodDeletion</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>BackupDoneBeforePodDeletion tells if backup before pod deletion has been made</p>
</td>
</tr>
<tr>
<td>
<code>userAndPasswordHash</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>UserAndPasswordHash is a SHA256 hash made from user and password</p>
</td>
</tr>
<tr>
<td>
<code>createdSeedJobs</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CreatedSeedJobs contains list of seed job id already created in Jenkins</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Plugin">Plugin
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsMaster">JenkinsMaster</a>)
</p>
<p>
<p>Plugin defines Jenkins plugin</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of Jenkins plugin</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the version of Jenkins plugin</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Restore">Restore
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsSpec">JenkinsSpec</a>)
</p>
<p>
<p>Restore defines configuration of Jenkins backup restore operation</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>containerName</code></br>
<em>
string
</em>
</td>
<td>
<p>ContainerName is the container name responsible for restore backup operation</p>
</td>
</tr>
<tr>
<td>
<code>action</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Handler">
Handler
</a>
</em>
</td>
<td>
<p>Action defines action which performs restore backup in restore container sidecar</p>
</td>
</tr>
<tr>
<td>
<code>recoveryOnce</code></br>
<em>
uint64
</em>
</td>
<td>
<em>(Optional)</em>
<p>RecoveryOnce if want to restore specific backup set this field and then Jenkins will be restarted and desired backup will be restored</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.SeedJob">SeedJob
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsSpec">JenkinsSpec</a>)
</p>
<p>
<p>SeedJob defines configuration for seed job
More info: <a href="https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-seed-jobs-and-pipelines">https://github.com/jenkinsci/kubernetes-operator/blob/master/docs/getting-started.md#configure-seed-jobs-and-pipelines</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the unique seed job name</p>
</td>
</tr>
<tr>
<td>
<code>credentialID</code></br>
<em>
string
</em>
</td>
<td>
<p>CredentialID is the Kubernetes secret name which stores repository access credentials</p>
</td>
</tr>
<tr>
<td>
<code>description</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Description is the description of the seed job</p>
</td>
</tr>
<tr>
<td>
<code>targets</code></br>
<em>
string
</em>
</td>
<td>
<p>Targets is the repository path where are seed job definitions</p>
</td>
</tr>
<tr>
<td>
<code>repositoryBranch</code></br>
<em>
string
</em>
</td>
<td>
<p>RepositoryBranch is the repository branch where are seed job definitions</p>
</td>
</tr>
<tr>
<td>
<code>repositoryUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>RepositoryURL is the repository access URL. Can be SSH or HTTPS.</p>
</td>
</tr>
<tr>
<td>
<code>credentialType</code></br>
<em>
<a href="#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.JenkinsCredentialType">
JenkinsCredentialType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>JenkinsCredentialType is the <a href="https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/">https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/</a> credential type</p>
</td>
</tr>
</tbody>
</table>
<h3 id="github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service">Service
</h3>
<p>
(<em>Appears on:</em>
<a href="#github.com%2fjenkinsci%2fkubernetes-operator%2fpkg%2fapis%2fjenkins%2fv1alpha2.JenkinsSpec">JenkinsSpec</a>)
</p>
<p>
<p>Service defines Kubernetes service attributes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations is an unstructured key value map stored with a resource that may be
set by external tools to store and retrieve arbitrary metadata. They are not
queryable and should be preserved when modifying objects.
More info: <a href="http://kubernetes.io/docs/user-guide/annotations">http://kubernetes.io/docs/user-guide/annotations</a></p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Route service traffic to pods with label keys and values matching this
selector. If empty or not present, the service is assumed to have an
external process managing its endpoints, which Kubernetes will not
modify. Only applies to types ClusterIP, NodePort, and LoadBalancer.
Ignored if type is ExternalName.
More info: <a href="https://kubernetes.io/docs/concepts/services-networking/service/">https://kubernetes.io/docs/concepts/services-networking/service/</a></p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.12/#servicetype-v1-core">
Kubernetes core/v1.ServiceType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Type determines how the Service is exposed. Defaults to ClusterIP. Valid
options are ExternalName, ClusterIP, NodePort, and LoadBalancer.
&ldquo;ExternalName&rdquo; maps to the specified externalName.
&ldquo;ClusterIP&rdquo; allocates a cluster-internal IP address for load-balancing to
endpoints. Endpoints are determined by the selector or if that is not
specified, by manual construction of an Endpoints object. If clusterIP is
&ldquo;None&rdquo;, no virtual IP is allocated and the endpoints are published as a
set of endpoints rather than a stable IP.
&ldquo;NodePort&rdquo; builds on ClusterIP and allocates a port on every node which
routes to the clusterIP.
&ldquo;LoadBalancer&rdquo; builds on NodePort and creates an
external load-balancer (if supported in the current cloud) which routes
to the clusterIP.
More info: <a href="https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services---service-types">https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services&mdash;service-types</a></p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
int32
</em>
</td>
<td>
<p>The port that are exposed by this service.
More info: <a href="https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies">https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies</a></p>
</td>
</tr>
<tr>
<td>
<code>nodePort</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>The port on each node on which this service is exposed when type=NodePort or LoadBalancer.
Usually assigned by the system. If specified, it will be allocated to the service
if unused or else creation of the service will fail.
Default is to auto-allocate a port if the ServiceType of this Service requires one.
More info: <a href="https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport">https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport</a></p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerSourceRanges</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>If specified and supported by the platform, this will restrict traffic through the cloud-provider
load-balancer will be restricted to the specified client IPs. This field will be ignored if the
cloud-provider does not support the feature.&rdquo;
More info: <a href="https://kubernetes.io/docs/tasks/access-application-cluster/configure-cloud-provider-firewall/">https://kubernetes.io/docs/tasks/access-application-cluster/configure-cloud-provider-firewall/</a></p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerIP</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Only applies to Service Type: LoadBalancer
LoadBalancer will get created with the IP specified in this field.
This feature depends on whether the underlying cloud-provider supports specifying
the loadBalancerIP when a load balancer is created.
This field will be ignored if the cloud-provider does not support the feature.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>37e531a</code>.
</em></p>
