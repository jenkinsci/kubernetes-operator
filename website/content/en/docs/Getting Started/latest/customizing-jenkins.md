---
title: "Customizing Jenkins"
linkTitle: "Customizing Jenkins"
weight: 3
date: 2021-12-08
description: >
  How to customize Jenkins
---

## How to customize Jenkins
Jenkins can be customized with plugins.
Plugin's configuration is applied as groovy scripts or the [configuration as code plugin](https://github.com/jenkinsci/configuration-as-code-plugin).
Any plugin working for Jenkins can be installed by the Jenkins Operator.

Pre-installed plugins:

* configuration-as-code v1.55
* git v4.10.0
* job-dsl v1.78.1
* kubernetes-credentials-provider v0.20
* kubernetes v1.30.11
* workflow-aggregator v2.6
* workflow-job v2.42

Rest of the plugins can be found in [plugins repository](https://plugins.jenkins.io/).


#### Install plugins

Edit Custom Resource under `spec.master.plugins`:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  master:
   plugins:
   - name: simple-theme-plugin
     version: "0.7"
```

Under `spec.master.basePlugins` you can find plugins for a valid **Jenkins Operator**:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  master:
    basePlugins:
    - name: kubernetes
      version: "1.30.11"
    - name: workflow-job
      version: "2.42"
    - name: workflow-aggregator
      version: "2.6"
    - name: git
      version: "4.10.0"
    - name: job-dsl
      version: "1.78.1"
    - name: configuration-as-code
      version: "1.55"
    - name: kubernetes-credentials-provider
      version: "0.20"
```

You can change their versions.

The **Jenkins Operator** will then automatically install plugins after the Jenkins master pod restart.

#### Apply plugin's config

By using a [ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) you can create your own **Jenkins** customized configuration.
Then you must reference the **`ConfigMap`** in the **Jenkins** pod customization file in `spec.groovyScripts` or `spec.configurationAsCode`

Create a **`ConfigMap`** with specific name (eg. `jenkins-operator-user-configuration`). Then, modify the **Jenkins** manifest:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  configurationAsCode:
    configurations:
    - name: jenkins-operator-user-configuration
  groovyScripts:
    configurations:
    - name: jenkins-operator-user-configuration
```

Here is an example of `jenkins-operator-user-configuration`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jenkins-operator-user-configuration
data:
  1-configure-theme.groovy: |
    import jenkins.*
    import jenkins.model.*
    import hudson.*
    import hudson.model.*
    import org.jenkinsci.plugins.simpletheme.ThemeElement
    import org.jenkinsci.plugins.simpletheme.CssTextThemeElement
    import org.jenkinsci.plugins.simpletheme.CssUrlThemeElement

    Jenkins jenkins = Jenkins.getInstance()

    def decorator = Jenkins.instance.getDescriptorByType(org.codefirst.SimpleThemeDecorator.class)

    List<ThemeElement> configElements = new ArrayList<>();
    configElements.add(new CssTextThemeElement("DEFAULT"));
    configElements.add(new CssUrlThemeElement("https://cdn.rawgit.com/afonsof/jenkins-material-theme/gh-pages/dist/material-light-green.css"));
    decorator.setElements(configElements);
    decorator.save();

    jenkins.save()
  1-system-message.yaml: |
    jenkins:
      systemMessage: "Configuration as Code integration works!!!"
```

* `*.groovy` is Groovy script configuration
* `*.yaml is` configuration as code

If you want to correct your configuration you can edit it while the **Jenkins Operator** is running.
Jenkins will reconcile and apply the new configuration.

## How to use secrets from a Groovy scripts

If you configured `spec.groovyScripts.secret.name`, then this secret is available to use from map Groovy scripts.
The secrets are loaded to `secrets` map.

Create a [secret](https://kubernetes.io/docs/concepts/configuration/secret/) with for example the name `jenkins-conf-secrets`.

```yaml
kind: Secret
apiVersion: v1
type: Opaque
metadata:
  name: jenkins-conf-secrets
  namespace: default
data:
  SYSTEM_MESSAGE: SGVsbG8gd29ybGQ=
```

Then modify the **Jenkins** pod manifest by changing `spec.groovyScripts.secret.name` to `jenkins-conf-secrets`.

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  configurationAsCode:
    configurations:
    - name: jenkins-operator-user-configuration
    secret:
      name: jenkins-conf-secrets
  groovyScripts:
    configurations:
    - name: jenkins-operator-user-configuration
    secret:
      name: jenkins-conf-secrets
```

Now you can test that the secret is mounted by applying this `ConfigMap` for Groovy script:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jenkins-operator-user-configuration
data:
  1-system-message.groovy: |
    import jenkins.*
    import jenkins.model.*
    import hudson.*
    import hudson.model.*
    Jenkins jenkins = Jenkins.getInstance()

    jenkins.setSystemMessage(secrets["SYSTEM_MESSAGE"])
    jenkins.save()
```

Or by applying this configuration as code:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jenkins-operator-user-configuration
data:
  1-system-message.yaml: |
    jenkins:
      systemMessage: ${SYSTEM_MESSAGE}
```


After this, you should see the `Hello world` system message from the **Jenkins** homepage.

## Externally managed pod fields (unreleased)

{{% pageinfo %}}
`spec.reconciliation.ignore` is available on current main after 0.9.0. It requires
an operator image and CRD built with this feature; the published 0.9.0 release does
not support it. Upgrade both before using these fields.
{{% /pageinfo %}}

### Built-in cloud identity rules {#built-in-cloud-identity-rules}

The maintained catalog is enabled by default:

```yaml
spec:
  reconciliation:
    ignoreWellKnownInjections: true
    ignore:
      env: [MY_CUSTOM_ENV]
      volumes: [my-custom-token]
```

Set the switch to `false` to disable the built-in rules. Manual lists are additive when enabled and work
independently when disabled. Helm uses
`jenkins.reconciliation.ignoreWellKnownInjections`. No provider selection is needed.

The following list is ignored **only when the switch is enabled**. No annotation
or label keys are included.

| Integration | Exact env names | Volume names |
| --- | --- | --- |
| AWS IRSA | `AWS_ROLE_ARN`, `AWS_WEB_IDENTITY_TOKEN_FILE`, `AWS_REGION`, `AWS_DEFAULT_REGION`, `AWS_STS_REGIONAL_ENDPOINTS` | `aws-iam-token` |
| EKS Pod Identity | `AWS_CONTAINER_CREDENTIALS_FULL_URI`, `AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE` | `eks-pod-identity-token` |
| Azure Workload Identity | `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_AUTHORITY_HOST`, `AZURE_FEDERATED_TOKEN_FILE` | `azure-identity-token`, or literal prefix `azure-workload-identity-reserved-` |
| Google ADC | `GOOGLE_APPLICATION_CREDENTIALS` | No standardized injected volume name; configure your injector's volume manually |

Matching volumes also ignore their mounts by volume name. These rules apply to both
desired and actual fields, **including matching names explicitly declared in the
CR**. For example, changing declared `AWS_REGION` will not cause recreation while
the switch is enabled. Use the manual lists with the switch disabled when you need
a narrower policy. Other `AWS_*`, `AZURE_*`, or `GOOGLE_*` names are not ignored.

Native GKE identity needs no such env/volume rules. Injector opt-in labels and
configuration annotations remain managed; ServiceAccount annotations are not pod
annotations. No blanket metadata-prefix rule is added: extra actual pod metadata
is already accepted. Istio, Linkerd, Azure proxy sidecars, arbitrary injected
containers and probe rewrites are outside this switch's scope. Failed pod,
credentials, operator version and other retained-field restart reasons still apply.

### Manual rules

Admission webhooks can inject identity environment variables and token volumes.
By default, changes to compared pod fields cause the operator to recreate Jenkins.
Declare only fields owned by your injector:

```yaml
spec:
  reconciliation:
    ignore:
      annotations: [identity.example.com/managed]
      labels: [identity.example.com/managed]
      env: [EXTERNAL_TOKEN_FILE]
      volumes: [external-identity-token]
      volumeNamePrefixes: []
```

All lists use literal, case-sensitive names or metadata keys. `env` ignores the
entire named variable, including `valueFrom`, in every compared regular container
(main and auxiliary); it does not ignore `envFrom`. A volume rule also ignores
mounts with that **volume name**, regardless of mount path. Both desired and
actual fields are filtered during comparison. A prefix is literal, not a glob or
regular expression; empty prefixes are rejected. Prefer exact names unless the
injector generates volume names.

Rules affect drift comparison only. They do not remove fields from pod creation,
relax validation, or prevent restarts for failed pods, changed credentials, operator
upgrades, plugins, images, probes, or other retained configuration. Extra actual pod
labels and annotations remain accepted without a rule; use a rule when an injector
changes or removes a key declared in `spec.master`. Extra containers are still drift.
The experimental `jenkins.io/use-deployment` path does not use these pod-recreation
rules. Init containers are outside the existing regular-container comparison.

In Helm, use `jenkins.reconciliation.ignore` with the same five lists. The existing
`jenkins.lifecycle` value continues to configure container lifecycle hooks.

### Cloud setup

For AWS IRSA, configure the role annotation on `spec.serviceAccount`. For Azure, see [AKS](../aks/). Both use the
[built-in list](#built-in-cloud-identity-rules).

Native GKE Workload Identity Federation needs no ignore rules. A custom Google
ADC injector can use the switch for `GOOGLE_APPLICATION_CREDENTIALS`; add its
volume name to `ignore.volumes` manually.

### Kubernetes volume defaults

Independently of ignore rules, an omitted `defaultMode` compares equal to the
Kubernetes default `0644` (`420`) for ConfigMap, Secret, Projected and DownwardAPI
volumes. This prevents a restart loop when the API server fills the default.
Explicitly different permissions, volume sources and retained mounts still cause
reconciliation. Script volumes with explicit executable permissions retain them.
