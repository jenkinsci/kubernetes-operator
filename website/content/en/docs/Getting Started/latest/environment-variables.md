---
title: "Environment Variables"
linkTitle: "Environment Variables"
weight: 6
date: 2021-07-01
description: >
    How to specify Jenkins environment variables
---

## JVM specific environment variables
The Operator sets default JAVA_OPTS for the Jenkins container:

    - XX:MinRAMPercentage=50.0 
    - XX:MaxRAMPercentage=80.0 
    - Djenkins.install.runSetupWizard=false 
    - Djava.awt.headless=true

They are only used if the user doesn't specify any other JAVA_OPTS. If you want to use your custom JAVA_OPTS, these won't
be applied. 

## Jenkins specific environment variables
You can also pass environment variables to Jenkins, but be careful, you may need to set the same variables in
JenkinsKubernetesAgent Custom Resource.
```yaml
apiVersion: jenkins.io/v1beta1
kind: Jenkins
metadata:
  name: example
spec:
  podSpec:
    containers:
      - name: jenkins-controller
        env:
          - name: JENKINS_OPTS
            value: --prefix=/jenkins
```

```yaml
apiVersion: jenkins.io/v1beta1
kind: JenkinsKubernetesAgent
metadata:
  name: operator-agent
labels:
  jenkins.io/jenkins: example
spec:
  podSpec:
    containers:
      - name: jnlp
        env:
          - name: JENKINS_OPTS
            value: --prefix=/jenkins
```

## HTTP Proxy for downloading plugins
To use forwarding proxy with an operator to download plugins you need to add the following environment variable to
Jenkins Custom Resource, e.g.:
```yaml
apiVersion: jenkins.io/v1beta1
kind: Jenkins
metadata:
  name: example
spec:
  podSpec:
    containers:
      - name: jenkins-controller
        env:
          - name: CURL_OPTIONS
            value: -L -x <proxy_url>
```
In `CURL_OPTIONS` var you can set additional arguments to curl command.
