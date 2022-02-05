---
title: "OpenShift"
linkTitle: "OpenShift"
weight: 20
date: 2020-04-29
description: >
    Additional configuration for OpenShift
---

## SecurityContext

OpenShift enforces Security Constraints Context (scc) when deploying an image.
By default, container images run in restricted scc which prevents from setting
a fixed user id to run with. You need to have ensure that you do not provide a
securityContext with a runAsUser and that your image does not use a hardcoded user.

```yaml
securityContext: {}
```

## OpenShift Jenkins image

OpenShift provides a pre-configured Jenkins image containing  3 openshift plugins for
jenkins (openshift-login-plugin, openshift-sync-plugin and openshift-client-plugin)
which allows better jenkins integration with kubernetes and OpenShift.

The OpenShift Jenkins image requires additional configuration to be fully enabled.

### Sample OpenShift CR
The following Custom Resource can be used to create a Jenkins instance using the  
OpenShift Jenkins image and sets values for:
- `image: 'quay.io/openshift/origin-jenkins:latest' : This is the OpenShift Jenkins image.

- serviceAccount: to allow oauth authentication to work, the service account needs
a specific annotation pointing to the route exposing the jenkins service. Here,
the route is named `jenkins-route`

- `OPENSHIFT_ENABLE_OAUTH` environment variable for the master container is set to true.

Here is a complete Jenkins CR allowing the deployment of the Jenkins OpenShift image.
```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  annotations:
    jenkins.io/openshift-mode: 'true'
  name: jenkins
spec:
  serviceAccount:
    annotations:
      serviceaccounts.openshift.io/oauth-redirectreference.jenkins: '{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"jenkins-route"}}'
  master:
    containers:
    - name: jenkins-master
      image: 'quay.io/openshift/origin-jenkins:latest'
      command:
      - /usr/bin/go-init
      - '-main'
      - /usr/libexec/s2i/run
      env:
      - name: OPENSHIFT_ENABLE_OAUTH
        value: 'true'
      - name: OPENSHIFT_ENABLE_REDIRECT_PROMPT
        value: 'true'
      - name: DISABLE_ADMINISTRATIVE_MONITORS
        value: 'false'
      - name: KUBERNETES_MASTER
        value: 'https://kubernetes.default:443'
      - name: KUBERNETES_TRUST_CERTIFICATES
        value: 'true'
      - name: JENKINS_SERVICE_NAME
        value: jenkins-operator-http-jenkins
      - name: JNLP_SERVICE_NAME
        value: jenkins-operator-slave-jenkins
      - name: JENKINS_UC_INSECURE
        value: 'false'
      - name: JENKINS_HOME
        value: /var/lib/jenkins
      - name: JAVA_OPTS
        value: >-
          -XX:+UnlockExperimentalVMOptions -XX:+UnlockExperimentalVMOptions
          -XX:+UseCGroupMemoryLimitForHeap -XX:MaxRAMFraction=1
          -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true
      imagePullPolicy: Always
  service:
    port: 8080
    type: ClusterIP
  slaveService:
    port: 50000
    type: ClusterIP
```

### OpenShift OAuth integration
The creation of a Route is required for the integraiton of Jenkins with
OpenShift oauth authentication. By default, the jenkins http service is named
`jenkins-operator-http-${jenkins-cr-name}`

```bash
oc create route edge jenkins-route --service=jenkins-operator-http-jenkins
```
Note: the route name (jenkins-route) must match the pointed route on the serviceaccount annotation.


After the creation of the Route. It can be used to navigate to the Jenkins Login Page and login with your Openshift Credentials.
