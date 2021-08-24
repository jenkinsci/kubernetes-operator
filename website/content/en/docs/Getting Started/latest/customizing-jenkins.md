---
title: "Customizing Jenkins"
linkTitle: "Customizing Jenkins"
weight: 3
date: 2021-08-20
description: >
  How to customize Jenkins
---

{{% pageinfo %}}
This document contains instructions on how to customize Jenkins instance with plugins, Groovy Scripts and Configuration as Code.
{{% /pageinfo %}}

## How Jenkins Customization works
Current configuration mechanism is based on Kubernetes Custom Resource which is automatically created during the
installation phase and then used as a customization file by the Operator.

Every time you want to customize Jenkins, you need to do that in code by modifying existing Custom Resource file.
Any manual changes from the web interface will be overridden by automation or after the Jenkins restart.

Sections below explain how to configure Jenkins using plugins, Groovy Scripts and Configuration as Code (CasC).

## How to customize Jenkins with plugins
Plugins configuration is applied as groovy scripts or the [configuration as code plugin](https://github.com/jenkinsci/configuration-as-code-plugin).
Any plugin working for Jenkins can be installed by the Jenkins Operator.
 
Pre-installed plugins and their versions can be found [here](https://github.com/jenkinsci/kubernetes-operator/blob/v0.7.0/pkg/plugins/base_plugins.go).

Rest of the plugins can be found in the [plugins repository](https://plugins.jenkins.io/). 

#### Customizing base plugins versions
Under `spec.basePlugins` you can find plugins for a valid **Jenkins Operator**:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
    basePlugins:
    - name: kubernetes
      version: "1.30.0"
    - name: workflow-job
      version: "2.40"
    - name: workflow-aggregator
      version: "2.6"
    - name: git
      version: "4.7.2"
    - name: job-dsl
      version: "1.77"
    - name: configuration-as-code
      version: "1.51"
    - name: kubernetes-credentials-provider
      version: "0.18-1"
```
Plugin versions shown here might not be up to date with the ones currently used by the Operator, that can be found [here](https://github.com/jenkinsci/kubernetes-operator/blob/v0.7.0/pkg/plugins/base_plugins.go).
You can change versions of pre-installed plugins by modifying your Jenkins Custom Resource's `spec.basePlugins` section.

#### Installing additional plugins
To install additional plugins, edit Jenkins Custom Resource under `spec.plugins`:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
   plugins:
   - name: simple-theme-plugin
     version: "0.6"
```

After applying modifications to Jenkins Custom Resource `spec.basePlugins` or `plugins` section, **Jenkins Operator**
will automatically install plugins after the Jenkins Controller pod restart.

## Customization via Groovy Scripts and CaSC yamls
Jenkins instance can be customized using Groovy scripts or Configuration as Code (thanks to pre-installed Configuration 
as Code plugin). CasC scripts are more readable and simpler to write, so they should be your default choice. However,
when something is not supported by the CasC plugin or for more complex and low-level configuration, Groovy scripts are better.
They allow you to use [Jenkins internal API].

You can find examples of configuring Jenkins in both of those ways below. The examples assume Operator and Jenkins are
both running, and Jenkins is deployed in the `jenkins` namespace.

## Customization of Jenkins with Groovy Scripts
The overall process of configuration can be divided into:

* Creating a Custom Resource JenkinsGroovyScript, which contains Groovy script you want to execute.
* Optionally creating a Kubernetes Secret if you need to store secrets like password or certificates.


### 1. Creating a Secret
In case we want to use confidential data, we have to start by creating a Secret resource to wrap it in.
Since it’s a secret, it would be a good idea to encode it. Running:

```bash
echo -n "Hello World" | base64
```

will produce the following output:

```
SGVsbG8gd29ybGQ=
```

which we can place in a Secret config file as the value of secrets key.

```yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: jenkins-conf-secrets
  namespace: jenkins
data:
  SYSTEM_MESSAGE: SGVsbG8gd29ybGQ=
```

To create the secret, save the yaml below to a file called `secret.yaml` and run:

```bash
kubectl apply -f secret.yaml
```

### 2. Creating JenkinsGroovyScript
We need to create a JenkinsGroovyScript config file containing the configuration we want to apply. In the data section
we can use Groovy scripts to write configuration code. Since the secret is already present in the Cluster before
JenkinsGroovyScript we can safely reference its value.

```yaml
apiVersion: jenkins.io/v1beta1
kind: JenkinsGroovyScript
metadata:
  name: groovy-script
  namespace: jenkins
  labels:
    jenkins.io/jenkins: jenkins
spec:
  secretRef:
    namespace: jenkins
    name: jenkins-conf-secrets
  data: |
    import jenkins.model.Jenkins

    def systemMessage = "Hello " + secrets

    Jenkins jenkins = Jenkins.getInstance()
    jenkins.setSystemMessage(systemMessage)
    jenkins.save()    
```

Save the yaml shown above as `jenkins-groovy-script.yaml` and run:

```bash
kubectl apply -f jenkins-groovy-script.yaml
```

This event will trigger Jenkins Groovy script reconcile loop, and our configuration will be applied automatically.
You will see a tiny “Hello World” on the main page.

## Customization of Jenkins with Configuration as Code (CasC)
The overall process of customization can be divided into:

* Creating a JenkinsConfigurationAsCode Custom Resource, which contains configuration code in data section.
* Optionally creating a Kubernetes Secret if you need to store secrets like passoword or certificates.
  
### 1. Creating a Secret
Creating a Secret for Configuration as Code works exactly like creating Secret for Groovy Scripts. Secret used in the
example JenkinsConfigurationAsCode can be created just like in Creating a Secret step of Customization of Jenkins with
Groovy Scripts section.

### 2. Creating JenkinsConfigurationAsCode 
We need to create a JekinsConfigurationAsCode config file containing the configuration we want to apply. In the data
field value we can use yaml syntax to add fields with configuration code. Since the secret is already present in the
Cluster before JekinsConfigurationAsCode we can safely reference its value.

```yaml
apiVersion: jenkins.io/v1beta1
kind: JenkinsConfigurationAsCode
metadata:
  name: jenkins-user-configuration
  namespace: jenkins
  labels:
    operator-service.com/jenkins: jenkins
spec:
  secretRef:
    namespace: jenkins
    name: jenkins-conf-secrets
data: |
    jenkins:
      systemMessage: ${SYSTEM_MESSAGE}
```

Save the yaml shown above as `jenkins-casc.yaml` and run:

```bash
kubectl -n jenkins apply -f jenkins-casc.yaml
```

Jenkins instance will see it and be able to bind it to previously created Secret, thanks to the reference in secretRef.
This event will trigger CasC reconcile loop, and our configuration will be applied. You will see a tiny "Hello World"
on the main page.

[Jenkins internal API]:https://javadoc.jenkins.io/