---
title: "Customization"
linkTitle: "Customization"
weight: 3
date: 2019-08-05
description: >
  How to customize Jenkins
---

Jenkins can be customized using groovy scripts or the [configuration as code plugin](https://github.com/jenkinsci/configuration-as-code-plugin). 
By using a [ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) you can create your own **Jenkins** customized configuration.
Then you must reference the **`ConfigMap`** in the **Jenkins** pod customization file in `spec.groovyScripts` or `spec.configurationAsCode`

For example create a **`ConfigMap`** with name `jenkins-operator-user-configuration`. Then, modify the **Jenkins** manifest to look like this:

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

### Using secrets from a Groovy script

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

## Install Plugins

Edit Custom Resource under `spec.master.plugins`:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  master:
   plugins:
   - name: simple-theme-plugin
     version: 0.5.1
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
      version: 1.18.3
    - name: workflow-job
      version: "2.34"
    - name: workflow-aggregator
      version: "2.6"
    - name: git
      version: 3.12.0
    - name: job-dsl
      version: "1.76"
    - name: configuration-as-code
      version: "1.29"
    - name: configuration-as-code-support
      version: "1.19"
    - name: kubernetes-credentials-provider
      version: 0.12.1
```

You can change their versions.

The **Jenkins Operator** will then automatically install plugins after the Jenkins master pod restarts.
