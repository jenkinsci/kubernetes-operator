---
title: "Customization"
linkTitle: "Customization"
weight: 3
date: 2019-08-05
description: >
  How to customize Jenkins
---

Jenkins can be customized using groovy scripts or the configuration as code plugin. All custom configuration is stored in
the **jenkins-operator-user-configuration-<cr_name>** ConfigMap which is automatically created by the **Jenkins Operator**.

The **Jenkins Operator** creates a **jenkins-operator-user-configuration-<cr_name>** secret where the user can store sensitive 
information used for custom configuration. If you have an entry in the secret named `PASSWORD` then you can use it in the 
Configuration as Plugin as `adminAddress: "${PASSWORD}"`.

```
kubectl get secret jenkins-operator-user-configuration-<cr_name> -o yaml

kind: Secret
apiVersion: v1
type: Opaque
metadata:
  name: jenkins-operator-user-configuration-<cr_name>
  namespace: default
data:
  SECRET_JENKINS_ADMIN_ADDRESS: YXNkZgo=

```

```
kubectl get configmap jenkins-operator-user-configuration-<cr_name> -o yaml

apiVersion: v1
data:
  1-configure-theme.groovy: |2
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
  1-system-message.yaml: |2
    jenkins:
      systemMessage: "Configuration as Code integration works!!!"
      adminAddress: "${SECRET_JENKINS_ADMIN_ADDRESS}"
kind: ConfigMap
metadata:
  name: jenkins-operator-user-configuration-<cr_name>
  namespace: default
``` 

When the **jenkins-operator-user-configuration-<cr_name>** ConfigMap is updated Jenkins automatically 
runs the **jenkins-operator-user-configuration** Jenkins Job which executes all scripts then
runs the **jenkins-operator-user-configuration-casc** Jenkins Job which applies the Configuration as Code configuration.

## Install Plugins

Edit CR under `spec.master.plugins`:

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

Then the **Jenkins Operator** will automatically install those plugins after the Jenkins master pod restart.
