---
title: "Developer Guide"
linkTitle: "Developer Guide"
weight: 5
date: 2023-01-16
description: >
  Jenkins Operator for developers
---

{{% pageinfo %}}
This document explains how to setup your development environment.
{{% /pageinfo %}}

## Prerequisites

- [operator_sdk][operator_sdk] version 1.3.0
- [git][git_tool]
- [go][go_tool] version 1.15.6
- goimports, golint, checkmake and staticcheck
- [minikube][minikube] version 1.21.0 (preferred Hypervisor - [virtualbox][virtualbox]) (automatically downloaded)
- [docker][docker_tool] version 17.03+

## Clone repository and download dependencies

```bash
git clone git@github.com:jenkinsci/kubernetes-operator.git
cd kubernetes-operator
make go-dependencies
```

## Build and run with a minikube

Start minikube instance configured for **Jenkins Operator**. Appropriate minikube version will be downloaded to bin folder.
```bash
make minikube-start
```
Next run **Jenkins Operator** locally. 
```bash
make run
```
Console output indicating readiness of this phase:
```bash
+ build
+ run
kubectl config use-context minikube
Switched to context "minikube".
Watching 'default' namespace
bin/manager --jenkins-api-hostname=192.168.99.252 --jenkins-api-port=0 --jenkins-api-use-nodeport=true --cluster-domain=cluster.local 
2021-02-08T14:14:45.263+0100    INFO    cmd     Version: v0.5.0
2021-02-08T14:14:45.263+0100    INFO    cmd     Git commit: 305dbeda-dirty-dirty
2021-02-08T14:14:45.264+0100    INFO    cmd     Go Version: go1.15.6
2021-02-08T14:14:45.264+0100    INFO    cmd     Go OS/Arch: darwin/amd64
2021-02-08T14:14:45.264+0100    INFO    cmd     Watch namespace: default
2021-02-08T14:14:45.592+0100    INFO    controller-runtime.metrics      metrics server is starting to listen    {"addr": "0.0.0.0:8383"}
2021-02-08T14:14:45.599+0100    INFO    cmd     starting manager
2021-02-08T14:14:45.599+0100    INFO    controller-runtime.manager      starting metrics server {"path": "/metrics"}
2021-02-08T14:14:45.599+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: jenkins.io/v1alpha2, Kind=Jenkins"}
2021-02-08T14:14:45.700+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: /, Kind="}
2021-02-08T14:14:45.800+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: /, Kind="}
2021-02-08T14:14:45.901+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: /, Kind="}
2021-02-08T14:14:46.003+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: core/v1, Kind=Secret"}
2021-02-08T14:14:46.004+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: core/v1, Kind=ConfigMap"}
2021-02-08T14:14:46.004+0100    INFO    controller-runtime.manager.controller.jenkins   Starting EventSource    {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "source": "kind source: jenkins.io/v1alpha2, Kind=Jenkins"}
2021-02-08T14:14:46.004+0100    INFO    controller-runtime.manager.controller.jenkins   Starting Controller     {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins"}
2021-02-08T14:14:46.004+0100    INFO    controller-runtime.manager.controller.jenkins   Starting workers        {"reconciler group": "jenkins.io", "reconciler kind": "Jenkins", "worker count": 1}

```
Lastly apply Jenkins Custom Resource to minikube cluster:
```bash
kubectl apply -f config/samples/jenkins.io_v1alpha2_jenkins.yaml

{"level":"info","ts":1612790690.875426,"logger":"controller-jenkins","msg":"Setting default Jenkins container command","cr":"jenkins-example"}
{"level":"info","ts":1612790690.8754492,"logger":"controller-jenkins","msg":"Setting default Jenkins container JAVA_OPTS environment variable","cr":"jenkins-example"}
{"level":"info","ts":1612790690.875456,"logger":"controller-jenkins","msg":"Setting default operator plugins","cr":"jenkins-example"}
{"level":"info","ts":1612790690.875463,"logger":"controller-jenkins","msg":"Setting default Jenkins master service","cr":"jenkins-example"}
{"level":"info","ts":1612790690.875467,"logger":"controller-jenkins","msg":"Setting default Jenkins slave service","cr":"jenkins-example"}
{"level":"info","ts":1612790690.881811,"logger":"controller-jenkins","msg":"*v1alpha2.Jenkins/jenkins-example has been updated","cr":"jenkins-example"}
{"level":"info","ts":1612790691.252834,"logger":"controller-jenkins","msg":"Creating a new Jenkins Master Pod default/jenkins-jenkins-example","cr":"jenkins-example"}
{"level":"info","ts":1612790691.322793,"logger":"controller-jenkins","msg":"Jenkins master pod restarted by operator:","cr":"jenkins-example"}
{"level":"info","ts":1612790691.322817,"logger":"controller-jenkins","msg":"Jenkins Operator version has changed, actual '' new 'v0.5.0'","cr":"jenkins-example"}
{"level":"info","ts":1612790691.3228202,"logger":"controller-jenkins","msg":"Jenkins CR has been replaced","cr":"jenkins-example"}
{"level":"info","ts":1612790695.8789551,"logger":"controller-jenkins","msg":"Creating a new Jenkins Master Pod default/jenkins-jenkins-example","cr":"jenkins-example"}
{"level":"warn","ts":1612790817.9423082,"logger":"controller-jenkins","msg":"Reconcile loop failed: couldn't init Jenkins API client: Get \"http://192.168.99.254:31998/api/json\": dial tcp 192.168.99.254:31998: connect: connection refused","cr":"jenkins-example"}
{"level":"warn","ts":1612790817.9998221,"logger":"controller-jenkins","msg":"Reconcile loop failed: couldn't init Jenkins API client: Get \"http://192.168.99.254:31998/api/json\": dial tcp 192.168.99.254:31998: connect: connection refused","cr":"jenkins-example"}
{"level":"info","ts":1612790818.581316,"logger":"controller-jenkins","msg":"base-groovy ConfigMap 'jenkins-operator-base-configuration-jenkins-example' name '1-basic-settings.groovy' running groovy script","cr":"jenkins-example"}
...
{"level":"info","ts":1612790820.9473379,"logger":"controller-jenkins","msg":"base-groovy ConfigMap 'jenkins-operator-base-configuration-jenkins-example' name '8-disable-job-dsl-script-approval.groovy' running groovy script","cr":"jenkins-example"}
{"level":"info","ts":1612790821.244055,"logger":"controller-jenkins","msg":"Base configuration phase is complete, took 2m6s","cr":"jenkins-example"}
{"level":"info","ts":1612790821.7953842,"logger":"controller-jenkins","msg":"Waiting for Seed Job Agent `seed-job-agent`...","cr":"jenkins-example"}
...

{"level":"info","ts":1612790851.843638,"logger":"controller-jenkins","msg":"Waiting for Seed Job Agent `seed-job-agent`...","cr":"jenkins-example"}
{"level":"info","ts":1612790853.489524,"logger":"controller-jenkins","msg":"User configuration phase is complete, took 2m38s","cr":"jenkins-example"}

Two log lines says that Jenkins Operator works correctly:
 
* `Base configuration phase is complete` - ensures manifests, Jenkins pod, Jenkins configuration and Jenkins API token  
* `User configuration phase is complete` - ensures Jenkins restore, backup and seed jobs along with user configuration 

> Details about base and user phase can be found [here](https://jenkinsci.github.io/kubernetes-operator/docs/how-it-works/architecture-and-design/).

```
```bash
kubectl get jenkins -o yaml

apiVersion: v1
items:
- apiVersion: jenkins.io/v1alpha2
  kind: Jenkins
  metadata:
  ...
  spec:
    backup:
      action: {}
      containerName: ""
      interval: 0
      makeBackupBeforePodDeletion: false
    configurationAsCode:
      configurations: []
      secret:
        name: ""
    groovyScripts:
      configurations: []
      secret:
        name: ""
    jenkinsAPISettings:
      authorizationStrategy: createUser
    master:
      basePlugins:
      ...
      containers:
      - command:
        - bash
        - -c
        - /var/jenkins/scripts/init.sh && exec /usr/bin/tini -s -- /usr/local/bin/jenkins.sh
        env:
        - name: JAVA_OPTS
          value: -XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap
            -XX:MaxRAMFraction=1 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true
        image: jenkins/jenkins:2.263.3-lts-alpine
        imagePullPolicy: Always
        livenessProbe:
        ...
        readinessProbe:
        ...
        resources:
          limits:
            cpu: 1500m
            memory: 3Gi
          requests:
            cpu: "1"
            memory: 500Mi
      disableCSRFProtection: false
    restore:
      action: {}
      containerName: ""
      getLatestAction: {}
    seedJobs:
    - additionalClasspath: ""
      bitbucketPushTrigger: false
      buildPeriodically: ""
      description: Jenkins Operator repository
      failOnMissingPlugin: false
      githubPushTrigger: false
      id: jenkins-operator
      ignoreMissingFiles: false
      pollSCM: ""
      repositoryBranch: master
      repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
      targets: cicd/jobs/*.jenkins
      unstableOnDeprecation: false
    service:
      port: 8080
      type: NodePort
    serviceAccount: {}
    slaveService:
      port: 50000
      type: ClusterIP
  status:
    appliedGroovyScripts:
    - configurationType: base-groovy
      hash: 2ownqpRyBjQYmzTRttUx7axok3CKe2E45frI5iRwH0w=
      name: 1-basic-settings.groovy
      source: jenkins-operator-base-configuration-jenkins-example
    ...
    baseConfigurationCompletedTime: "2021-02-08T13:27:01Z"
    createdSeedJobs:
    - jenkins-operator
    operatorVersion: v0.5.0
    provisionStartTime: "2021-02-08T13:24:55Z"
    userAndPasswordHash: nnfZsWmFfAYlYyVYeKhWW2KB4L8mE61JUfetAsr9IMM=
    userConfigurationCompletedTime: "2021-02-08T13:27:33Z"
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

```bash
kubectl get po

NAME                                              READY   STATUS              RESTARTS   AGE
jenkins-jenkins-example                           1/1     Running             0          23m
seed-job-agent-jenkins-example-758cc7cc5c-82hbl   1/1     Running             0          21m

```

## Build and run with Docker Desktop

Install Docker Desktop. If you are using Docker Desktop for Windows, you will also need to install WSL or WSL2. Ensure that Docker Desktop is currently running, and that you have enabled Kubernetes in it.

Run **Jenkins Operator** locally. 
```bash
make config="config.docker-desktop.env" run
```

From this point on, Docker Desktop usage is identical to minikube usage.

### Debug Jenkins Operator

```bash
make run OPERATOR_EXTRA_ARGS="--debug"
```

### Stop or delete minikube cluster
To stop Kubernetes cluster running locally on minikube:
```bash
minikube stop
```
To delete the cluster altogether:
```bash
minikube delete
```

## Build and run with a remote Kubernetes cluster

You can also run the controller locally and make it listen to a remote Kubernetes server.

```bash
make run NAMESPACE=default KUBECTL_CONTEXT=remote-k8s EXTRA_ARGS='--kubeconfig ~/.kube/config'
```

Once **Jenkins Operator** are up and running, apply Jenkins custom resource:

```bash
kubectl --context remote-k8s --namespace default apply -f deploy/crds/jenkins_v1alpha2_jenkins_cr.yaml
kubectl --context remote-k8s --namespace default get jenkins -o yaml
kubectl --context remote-k8s --namespace default get po
```

## Testing

Tests are written using [Ginkgo](https://onsi.github.io/ginkgo/) with [Gomega](https://onsi.github.io/gomega/). 

Run unit tests with go fmt, lint, staticcheck, vet:

```bash
make verify
```

Run unit tests only:

```bash
make test
```

### Running E2E tests

Run e2e tests with minikube:

```bash
make minikube-start
make e2e
```

Run Helm e2e tests:
```bash
eval $(bin/minikube docker-env)
make helm-e2e
```

Run the specific e2e test:

```bash
make e2e E2E_TEST_SELECTOR='^TestConfiguration$'
```

### Building docker image on minikube

To be able to work with the docker daemon on `minikube` machine run the following command before building an image:

```bash
eval $(bin/minikube docker-env)
```

### When `api/v1alpha2/jenkins_types.go` has changed

Run:

```bash
make manifests
```

### Getting the Jenkins URL and basic credentials

```bash
minikube service jenkins-operator-http-<cr_name> --url
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.password}' | base64 -d
```

### Webhook
To deploy the operator along with webhook, run :
```bash
eval $(minikube docker-env)
make deploy-webhook
```
It uses [cert-manager](https://cert-manager.io/) as an external dependency.

## Self-learning

* [Tutorial: Deep Dive into the Operator Framework for... Melvin Hillsman, Michael Hrivnak, & Matt Dorn
](https://www.youtube.com/watch?v=8_DaCcRMp5I)

* [Operator Framework Training By OpenShift](https://www.katacoda.com/openshift/courses/operatorframework)

* [Operator SDK Tutorial for Go](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)

* [Kubebuilder Validating Webhook Implementation](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html)

[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[operator_sdk]:https://github.com/operator-framework/operator-sdk
[fork_guide]:https://help.github.com/articles/fork-a-repo/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube]:https://kubernetes.io/docs/tasks/tools/install-minikube/
[virtualbox]:https://www.virtualbox.org/wiki/Downloads
