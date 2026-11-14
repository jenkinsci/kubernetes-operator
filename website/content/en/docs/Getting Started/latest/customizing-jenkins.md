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

## Seed agent scheduling (unreleased)

The seed agent accepts only jobs targeting `seed-job-agent` by default, including
existing agents after upgrade. Set `spec.seedJobRestrictJobsToLabel: false` (Helm:
`jenkins.seedJobRestrictJobsToLabel: false`) to allow unlabeled jobs.

## Jenkins home storage (unreleased)

Jenkins home uses `emptyDir` by default. See [why Jenkins home is disposable]({{< relref "/docs/FAQ/_index.md#how-can-i-change-jenkins_home-from-volume-to-persistent-volume" >}}).
To use a Pod-owned ephemeral PVC:

```yaml
spec:
  master:
    homeVolume:
      ephemeral:
        storageClassName: fast
        accessModes: [ReadWriteOnce]
        resources:
          requests:
            storage: 20Gi
```

Select one source: `ephemeral` or `emptyDir` (optional `medium` and `sizeLimit`).
Omit `storageClassName` for the cluster default; `""` requests no StorageClass.
Helm uses `jenkins.homeVolume`.

Changing home storage replaces the Pod without migrating data. Existing
backup and restore settings still apply.

## Externally managed pod fields (unreleased)

{{% pageinfo %}}
These options require an operator image and CRD built after 0.9.0; the published
0.9.0 release does not support them.
{{% /pageinfo %}}

### Built-in cloud identity rules {#built-in-cloud-identity-rules}

`spec.reconciliation.ignoreWellKnownInjections` defaults to `true`. It prevents
the following injected fields from triggering pod recreation. Set it to `false`
to disable these rules.

| Integration | Exact env names | Volume names |
| --- | --- | --- |
| AWS IRSA | `AWS_ROLE_ARN`, `AWS_WEB_IDENTITY_TOKEN_FILE`, `AWS_REGION`, `AWS_DEFAULT_REGION`, `AWS_STS_REGIONAL_ENDPOINTS` | `aws-iam-token` |
| EKS Pod Identity | `AWS_CONTAINER_CREDENTIALS_FULL_URI`, `AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE` | `eks-pod-identity-token` |
| Azure Workload Identity | `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_AUTHORITY_HOST`, `AZURE_FEDERATED_TOKEN_FILE` | `azure-identity-token`, or prefix `azure-workload-identity-reserved-` |
| Google ADC | `GOOGLE_APPLICATION_CREDENTIALS` | Add your injector's volume name manually |

Matching volume mounts are also ignored. **Matching names explicitly declared in
your CR are ignored too**: changing `AWS_REGION`, for example, will not recreate
the pod. Disable the switch and use manual rules if you need narrower control.

Cloud identity configuration is still required; see [AKS](../aks/) for Azure.
Native GKE identity needs no ignore rules. No label or annotation keys are included
in the built-in list.

### Manual rules

Add rules for other externally managed fields. They work with or without the
built-in rules:

```yaml
spec:
  reconciliation:
    ignore:
      annotations: [identity.example.com/managed]
      labels: [identity.example.com/managed]
      env: [EXTERNAL_TOKEN_FILE]
      volumes: [external-identity-token]
      volumeNamePrefixes: [custom-token-]
```

Names and keys are exact and case-sensitive. Volume prefixes are literal, non-empty
strings, not globs; volume rules also ignore mounts by volume name. Env rules cover
`value` and `valueFrom` in regular containers, but not `envFrom` or init containers.
Rules apply to both desired and actual fields during comparison only.

Extra pod labels and annotations are already tolerated. Use manual metadata rules
when another controller changes or removes keys declared in `spec.master`.
ServiceAccount annotations remain managed separately.

Other restart checks remain active, including failures, credentials, operator
upgrades and changes to retained fields. Unrecognized sidecars and probe rewrites
are not covered. The experimental Deployment path does not use these rules.

Helm uses the same settings under `jenkins.reconciliation`.

Istio and Linkerd regular/native sidecars and their known injected volumes are
covered by the same switch; [exact component names](https://github.com/jenkinsci/kubernetes-operator/blob/master/pkg/configuration/base/mesh_comparison.go).
Mesh rules preserve user-declared containers and volumes. Istio requires this Pod
annotation to keep application probes unchanged:

```yaml
spec:
  master:
    annotations:
      sidecar.istio.io/rewriteAppHTTPProbers: "false"
```

Full strict-mTLS compatibility is not supported: mesh policy must permit direct
kubelet probes and operator-to-Jenkins traffic. Linkerd does not use the Istio
annotation. See [Istio health checks](https://istio.io/latest/docs/ops/configuration/mesh/app-health-check/).

### Kubernetes volume defaults

For ConfigMap, Secret, Projected and DownwardAPI volumes, omitted `defaultMode`
compares equal to Kubernetes' default `0644` (`420`), independently of ignore rules.
Different permissions or volume sources still trigger reconciliation.
