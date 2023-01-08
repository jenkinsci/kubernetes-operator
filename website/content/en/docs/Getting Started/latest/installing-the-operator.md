---
title: "Installing the Operator"
linkTitle: "Installing the Operator"
weight: 1
date: 2023-01-08
description: >
  How to install Jenkins Operator
---

{{% pageinfo %}}
This document describes installation procedure for **Jenkins Operator**. 
All container images can be found at [virtuslab/jenkins-operator](https://hub.docker.com/r/virtuslab/jenkins-operator) Docker Hub repository.
{{% /pageinfo %}}

## Requirements
 
To run **Jenkins Operator**, you will need:

- access to a Kubernetes cluster version `1.17+`
- `kubectl` version `1.17+`


Listed below are the two ways to deploy Jenkins Operator.

## Deploy Jenkins Operator using YAML's

First, install Jenkins Custom Resource Definition:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/config/crd/bases/jenkins.io_jenkins.yaml 
```

Then, install the Operator and other required resources:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/all-in-one-v1alpha2.yaml
```

Watch **Jenkins Operator** instance being created:

```bash
kubectl get pods -w
```

Now **Jenkins Operator** should be up and running in the `default` namespace.
For deploying Jenkins, refer to [Deploy Jenkins section](/kubernetes-operator/docs/getting-started/latest/deploying-jenkins/).

## Deploy Jenkins Operator using Helm Chart

Alternatively, you can also use Helm to install the Operator (and optionally, by default, Jenkins). It requires the Helm 3+ for deployment.

Create a namespace for the operator:

```bash
$ kubectl create namespace <your-namespace>
```

To install, you need only to type these commands:

```bash
$ helm repo add jenkins https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart
$ helm install <name> jenkins/jenkins-operator -n <your-namespace>
```

To add custom labels and annotations, you can use `values.yaml` file or pass them into `helm install` command, e.g.:

```bash
$ helm install <name> jenkins/jenkins-operator -n <your-namespace> --set jenkins.labels.LabelKey=LabelValue,jenkins.annotations.AnnotationKey=AnnotationValue
```
You can further customize Jenkins using `values.yaml`:
<h3 id="JenkinsConfiguration">Jenkins instance configuration
</h3>

<table aria-colspan="4">
<thead aria-colspan="4">
<tr>
<th></th>
<th>Field</th>
<th>Default value</th>
<th>Description</th>
</tr>
</thead>
<tbody aria-colspan="4">
<tr></tr>
<tr>
<td colspan="1">
<code>jenkins</code>
</td>
<td colspan="3">
<p>operator is section for configuring operator deployment</p>
<table>
<tr>
<td>
<code>enabled</code>
</td>
<td>
true
</td>
<td>
Enabled can enable or disable the Jenkins instance. 
Set to false if you have configured CR already and/or you want to deploy an operator only.
</td>
</tr>
<tr>
<td>
<code>apiVersion</code>
</td>
<td>jenkins.io/v1alpha2</td>
<td>
Version of the CR manifest. The recommended and default value is <code>jenkins.io/v1alpha2</code>.
<a href="#github.io/kubernetes-operator/docs/getting-started/v0.1.x/migration-guide-v1alpha1-to-v1alpha2/">More info</a>
</td>
</tr>
<tr>
<td>
<code>name</code>
</td>
<td>
jenkins
</td>
<td>
Name of resource. The pod name will be <code>jenkins-&lt;name&gt;</code> (name will be set as suffix).
</td>
</tr>
<tr>
<td>
<code>namespace</code>
</td>
<td>
default
</td>
<td>
Namespace the resources will be deployed to. It's not recommended to use default namespace. 
Create new namespace for jenkins (e.g. <code>kubectl create -n jenkins</code>)
</td>
</tr>
<tr>
<td>
<code>labels</code>
</td>
<td>
{}
</td>
<td>
Labels are injected into metadata labels field.
</td>
</tr>
<tr>
<td>
<code>annotations</code>
</td>
<td>
{}
</td>
<td>
Annotations are injected into metadata annotations field.
</td>
</tr>
<tr>
<td>
<code>image</code>
</td>
<td>
jenkins/jenkins:lts
</td>
<td>
Image is the name (and tag) of the Jenkins instance.
It's recommended to use LTS (tag: "lts") version.
</td>
</tr>
<tr>
<td>
<code>env</code>
</td>
<td>
[]
</td>
<td>
Env contains jenkins container environment variables.
</td>
</tr>
<tr>
<td>
<code>imagePullPolicy</code>
</td>
<td>
Always
</td>
<td>
Defines policy for pulling images
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code>
</td>
<td>
""
</td>
<td>
PriorityClassName indicates the importance of a Pod relative to other Pods.
<a href="https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/">More info</a>
</td>
</tr>
<tr>
<td>
<code>disableCSRFProtection</code>
</td>
<td>
false
</td>
<td>
disableCSRFProtection can enable or disable operator built-in CSRF protection.
Set it to true if you are using OpenShift Jenkins Plugin.
<a href="https://github.com/jenkinsci/kubernetes-operator/pull/193">More info</a>
</td>
</tr>
<tr>
<td>
<code>imagePullSecrets</code>
</td>
<td>
[]
</td>
<td>
Used if you want to pull images from private repository
<a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuration/#pulling-docker-images-from-private-repositories">More info</a>
</td>
</tr>
<tr>
<td>
<code>notifications</code>
</td>
<td>
[]
</td>
<td>
Notifications is feature that notify user about Jenkins reconcilation status
<a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/notifications/">More info</a>  
</td>
</tr>
<tr>
<td>
<code>basePlugins</code>
</td>
<td>
<pre>
- name: kubernetes
  version: "1.25.2"
- name: workflow-job
  version: "2.39"
- name: workflow-aggregator
  version: "2.6"
- name: git
  version: "4.2.2"
- name: job-dsl
  version: "1.77"
- name: configuration-as-code
  version: "1.38"
- name: kubernetes-credentials
        -provider
  version: "0.13"
</pre>
</td>
<td>
Plugins installed and required by the operator
shouldn't contain plugins defined by user
You can change their versions here
<a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/customization/#install-plugins">More info</a>
</td>
</tr>
<tr>
<td>
<code>plugins</code>
</td>
<td>
[]
</td>
<td>
Plugins required by the user. You can define plugins here.
<a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/customization/#install-plugins">More info</a>
Example:
<pre>
plugins:
 - name: simple-theme-plugin
   version: 0.5.1
</pre>  
</td>
</tr>
<tr>
<td>
<code>seedJobs</code>
</td>
<td>
[]
</td>
<td>
Placeholder for jenkins seed jobs
For seed job creation tutorial, check:<br /> <a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-seed-jobs-and-pipelines/#prepare-job-definitions-and-pipelines">Prepare seed jobs</a>
<br /><a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-seed-jobs-and-pipelines/#configure-seed-jobs">Configure seed jobs</a>
<br />Example:
<code>
<pre>
seedJobs:
- id: jenkins-operator
  targets: "cicd/jobs/*.jenkins"
  description: "Jenkins Operator repository"
  repositoryBranch: master
  repositoryUrl: 
  - https://github.com/jenkinsci/kubernetes-operator.git
</pre>
</code>  
</td>
</tr>
<tr>
<td>
<code>resources</code>
</td>
<td>
<pre>
limits:
  cpu: 1500m
  memory: 3Gi
requests:
  cpu: 1
  memory: 500M
</pre>
</td>
<td>
Resource limit/request for Jenkins
<a href="https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container">More info</a>
</td>
</tr>
<tr>
<td>
<code>volumes</code>
</td>
<td>
<pre>
- name: backup
  persistentVolumeClaim:
    claimName: jenkins-backup
</pre>
</td>
<td>
Volumes used by Jenkins
By default, we are only using PVC volume for storing backups.
</td>
</tr>
<tr>
<td>
<code>volumeMounts</code>
</td>
<td>
[]
</td>
<td>
volumeMounts are mounts for Jenkins pod.
</td>
</tr>
<tr>
<td>
<code>securityContext</code>
</td>
<td>
runAsUser: 1000
fsGroup: 1000
</td>
<td>
SecurityContext for pod.
</td>
</tr>
<tr>
<td><code>service</code></td>
<td>not implemented</td>
<td>Http Jenkins service. See https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/schema/#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service for details.</td>
</tr>
<tr>
<td><code>slaveService</code></td>
<td>not implemented</td>
<td>Slave Jenkins service. See https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/schema/#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.Service for details.</td>
</tr>
<tr>
<td>
<code>livenessProbe</code>
</td>
<td>
<pre>
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
</pre>
</td>
<td>
livenessProbe for Pod
</td>
</tr>
<tr>
<td>
<code>readinessProbe</code>
</td>
<td>
<pre>
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
</pre>
</td>
<td>
readinessProbe for Pod
</td>
</tr>
<tr>
<td>
<code>
backup
</code>
<p>
<em>
<a href="#Backup">
Backup
</a>
</em>
</p>
</td>
<td>
</td>
<td>
Backup is section for configuring operator's backup feature
By default backup feature is enabled and pre-configured
This section simplifies the configuration described here: <a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configuring-backup-and-restore/">Configuring backup and restore</a>
For customization tips see <a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/custom-backup-and-restore">Custom backup and restore</a>
</td>
</tr>
<tr>
<td>
<code>configuration</code>
<p>
<em>
<a href="#Configuration">
Configuration
</a>
</em>
</p>
</td>
<td></td>
<td>
Section where we can configure Jenkins instance. 
See <a href="https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/customizing-jenkins/">Customizing Jenkins</a> for details
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>

### Configuring operator deployment

<table aria-colspan="4">
    <thead aria-colspan="4">
        <tr>
            <th></th>
            <th>Field</th>
            <th>Default value</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody aria-colspan="4">
        <tr></tr>
        <tr>
            <td colspan="1">
            <code>operator</code>
            </td>
            <td colspan="3">
            <p>operator is section for configuring operator deployment</p>
                <table>
                <tr>
                <td>
                <code>replicaCount</code></br>
                </td>
                <td>
                1
                </td>
                <td>
                Number of Replicas.
                </td>
                </tr>
                <tr>
                <td>
                <code>image</code>
                </td>
                <td>
                virtuslab/jenkins-operator:v0.4.0
                </td>
                <td>
                Name (and tag) of the Jenkins Operator image.
                </td>
                </tr>
                <tr>
                <td>
                <code>imagePullPolicy</code>
                </td>
                <td>
                IfNotPresent
                </td>
                <td>
                Defines policy for pulling images.
                </td>
                </tr>
                <tr>
                <td>
                <code>imagePullSecrets</code>
                </td>
                <td>
                []
                </td>
                <td>
                Used if you want to pull images from private repository.
                </td>
                </tr>
                <tr>
                <td>
                <code>nameOverride</code>
                </td>
                <td>
                ""
                </td>
                <td>
                nameOverride overrides the app name.
                </td>
                </tr>
                <tr>
                <td>
                <code>fullnameOverride</code>
                </td>
                <td>
                ""
                </td>
                <td>
                fullnameOverride overrides the deployment name
                </td>
                </tr>
                <tr>
                    <td>
                    <code>resources</code>
                    </td>
                    <td>
                    {}
                    </td>
                    <td>
                    </td>
                </tr>
                <tr>
                    <td>
                    <code>nodeSelector</code>
                    </td>
                    <td>
                    {}
                    </td>
                    <td>
                    </td>
                </tr>
                <tr>
                    <td>
                    <code>tolerations</code>
                    </td>
                    <td>
                    {}
                    </td>
                    <td>
                    </td>
                </tr>
                <tr>
                    <td>
                    <code>affinity</code>
                    </td>
                    <td>
                    {}
                    </td>
                    <td>
                    </td>
                </tr>
                </table>
            </td>
        </tr>
    </tbody>
</table>



<h3 id="Backup">Backup
</h3>
<p>
(<em>Appears on:</em>
<a href="#JenkinsConfiguration">JenkinsConfiguration</a>)
</p>
<p>
Backup defines configuration of Jenkins backup.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Default value</th>
<th>Description</th>
</tr>
</thead>
    <tbody>
        <tr>
            <td>
                <code>enabled</code>
            </td>
            <td>
                true
            </td>
            <td>
                Enabled is enable/disable switch for backup feature.
            </td>
        </tr>
        <tr>
            <td>
                <code>image</code>
            </td>
            <td>
                virtuslab/jenkins-operator-backup-pvc:v0.1.1
            </td>
            <td>
                Image used by backup feature.
            </td>
        </tr>
        <tr>
            <td>
                <code>containerName</code>
            </td>
            <td>
                backup
            </td>
            <td>
                Backup container name.
            </td>
        </tr>
        <tr>
            <td>
                <code>interval</code>
            </td>
            <td>
                30
            </td>
            <td>
                Defines how often make backup in seconds.
            </td>
        </tr>
        <tr>
            <td>
                <code>makeBackupBeforePodDeletion</code>
            </td>
            <td>
                true
            </td>
            <td>
                When enabled will make backup before pod deletion.
            </td>
        </tr>
        <tr>
            <td>
                <code>backupCommand</code>
            </td>
            <td>
                /home/user/bin/backup.sh
            </td>
            <td>
                Backup container command.
            </td>
        </tr>
        <tr>
            <td>
                <code>restoreCommand</code>
            </td>
            <td>
                /home/user/bin/restore.sh
            </td>
            <td>
                Backup restore command.
            </td>
        </tr>
        <tr>
            <td>
                <code>pvc</code>
            </td>                     
            <td colspan="2">
                <p>Persistent Volume Claim Kubernetes resource</p>
                <br/>
                <table colspan="2" style="width:100%">
                <tbody>
                    <tr>
                       <td>
                            <code>enabled</code>
                        </td>
                        <td>
                            true
                        </td>
                        <td>
                            Enable/disable switch for PVC                        
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <code>enabled</code>
                        </td>
                        <td>
                            true
                        </td>
                        <td>
                            Enable/disable switch for PVC
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <code>size</code>
                        </td>
                        <td>
                            5Gi
                        </td>
                        <td>
                            Size of PVC
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <code>className</code>
                        </td>
                        <td>
                            ""
                        </td>
                        <td>
                            StorageClassName for PVC  
                            <a href="https://kubernetes.io/docs/concepts/storage/persistent-volumes/#class-1">More info</a>
                        </td>
                    </tr>
                    </tbody>
                </table>
            </td>
        </tr>
         <tr>
            <td>
                <code>env</code>
            </td>          
            <td>
<pre>
- name: BACKUP_DIR
  value: /backup
- name: JENKINS_HOME
  value: /jenkins-home
- name: BACKUP_COUNT
  value: "3"
</pre>
            </td>
                <td>
                    Contains container environment variables. 
                    PVC backup provider handles these variables:<br />
                    BACKUP_DIR - path for storing backup files (default: "/backup")<br />
                    JENKINS_HOME - path to jenkins home (default: "/jenkins-home")<br />
                    BACKUP_COUNT - define how much recent backups will be kept<br />
                </td>
            </td>
        </tr>
        <tr>
        <td>
            <code>volumeMounts</code>
        </td>
        <td>
<pre>
- name: jenkins-home
  mountPath: /jenkins-home
- mountPath: /backup
  name: backup
</pre>
        </td>
        <td>
            Holds the mount points for volumes.
        </td>
        </tr>
    </tbody>
</table>
 
 <h4 id="Configuration">Configuration
 </h3>
 <p>
 (<em>Appears on:</em>
 <a href="#JenkinsConfiguration">Jenkins instance configuration</a>)
 </p>

 <table>
 <thead>
 <tr>
 <th>Field</th>
 <th>Default value</th>
 <th>Description</th>
 </tr>
 </thead>
     <tbody>
         <tr>
             <td>
                 <code>configurationAsCode</code>
             </td>
             <td>
                 {}
             </td>
             <td>
             ConfigurationAsCode defines configuration of Jenkins customization via Configuration as Code Jenkins plugin.
Example:<br />
<pre>
- configMapName: jenkins-casc
  content: {}
</pre>
             </td>
         </tr>
         <tr>
             <td>
                 <code>groovyScripts</code>
             </td>
             <td>
                 {}
             </td>
             <td>
             GroovyScripts defines configuration of Jenkins customization via groovy scripts.
             Example:<br />
<pre>
- configMapName: jenkins-gs
  content: {}
</pre>
             </td>
         </tr>
         <tr>
             <td>
                 <code>secretRefName</code>
             </td>
             <td>
                 ""
             </td>
             <td>
                 secretRefName of existing secret (previously created).
             </td>
         </tr>
         <tr>
             <td>
                 <code>secretData</code>
             </td>
             <td>
                 {}
             </td>
             <td>
                 If secretRefName is empty, secretData creates new secret and fills with data provided in secretData.
             </td>
         </tr>
     </tbody>
 </table>

## Note on Operator's nightly built images
If you wish to use the newest, not yet released version of the Operator, you can use one of nightly built snapshot images, however the maintainers of this project cannot guarantee their stability.

You can find nightly built images by heading to [virtuslab/jenkins-operator](https://hub.docker.com/r/virtuslab/jenkins-operator) Docker Hub repository and looking for images with tag in the form of `{git-hash}`, {git-hash} being the hash of master branch commit that you want to use snapshot of.

## Note on restricted Jenkins controller pod volumeMounts
Current design of the Operator puts an emphasis on creating a full GitOps flow of work for Jenkins users.
One of the key points of this design is maintaining an immutable state of Jenkins. 

One of the prerequisites of this is an ephemeral Jenkins home directory. To achieve that, Operator mounts emptyDir Volume
(jenkins-home) as Jenkins home directory.
It is not possible to overwrite volumeMount and specify any other Volume for Jenkins home directory,
as attempting to do so will result in Operator error.

jenkins-home is not the only Jenkins controller pod volumeMount that is non-configurable and managed by Operator,
below is the full list of those volumeMounts:

* jenkins-home
* scripts
* init-configuration
* operator-credentials

## Validating Webhook 
Validating webhook can be used in order to increase the Operator's capabilities to monitor security issues. It will look for security vulnerabilities in the base and requested plugins. It can be easily installed via Helm charts by setting webhook.enabled in values.yaml.


**Note**: The webhook takes some time to get up and running. It's recommended to first deploy the Operator and later Jenkins Custom Resource by using toggles in `values.yaml`.
For the installation with yaml manifests (without using Helm chart), first, install cert-manager:

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.5.1/cert-manager.yaml 
```

It takes some time to get cert-manager up and running.
Then, install the webhook and other required resources:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/all-in-one-webhook.yaml
```

Now, download the manifests for the operator and other resources from [here](https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/all-in-one-v1alpha2.yaml) and provide these additional fields in the Operator manifest:

<pre>
<code>
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jenkins-operator
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: jenkins-operator
      securityContext:
        runAsUser: 65532
      containers:
      - command:
        - /manager
        args:
        - --leader-elect
        <b>- --validate-security-warnings</b>
        image: virtuslab/jenkins-operator:v0.7.0
        name: jenkins-operator
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          limits:
            cpu: 200m
            memory: 100Mi
          requests:
            cpu: 100m
            memory: 20Mi
        env:
          - name: WATCH_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        <b>volumeMounts:
          - mountPath: /tmp/k8s-webhook-server/serving-certs
            name: webhook-certs
            readOnly: true       
      volumes:
      - name: webhook-certs
        secret:
          defaultMode: 420
          secretName: jenkins-webhook-certificate
      terminationGracePeriodSeconds: 10</b>
</code>
</pre>

To enable security validation in the Jenkins Custom Resource, set

>jenkins.ValidateSecurityWarnings=true

