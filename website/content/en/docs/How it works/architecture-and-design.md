---
title: "Architecture and design"
linkTitle: "Architecture and design"
weight: 1
date: 2019-08-05
description: >
  Jenkins Operator fundamentals
---

**Jenkins Operator** design incorporates the following concepts:

- watching any changes to manifests and maintaining the desired state according to deployed custom resource manifests
- implementing multiple reconciliation loops that ensure state of particular resources match the desired state defined in the manifests.

**Jenkins Operator** supports the following manifests:

* Jenkins
  * allows you to control options specific to the Jenkins Controller (Jenkins Master) as well as Kubernetes-specific options such as Roles or PodSpec,
  * it uses an InitContainer to ensure initial configuration,
  * it provides a mechanism for caching plugins, so when the pod restarts, it won't have to download them again.

- JenkinsKubernetesAgent
  - allows you to control options specific to the Jenkins Agents as well as Kubernetes-specific options such as Roles or PodSpec,
  - allows you to define agents using custom images,
  - a seedJob agent is required when using JenkinsSeedJobs, if it isn't present, one will be created by the operator.


- JenkinsSeedJob
  - uses Jenkins [Job DSL](https://plugins.jenkins.io/job-dsl/) plugin,
  - it is a recommended way to define your jobs,
  - jobs defined as JenkinsSeedJobs will be recreated on Jenkins restart.

- JenkinsConfigurationAsCode
  - uses Jenkins [Configuration as Code](https://plugins.jenkins.io/configuration-as-code/) plugin,
  - allows you to control the Jenkins' state directly, they will be re-applied on Jenkins restart.

- JenkinsGroovyScript
  - allows you to control the Jenkins' state directly, they will be re-applied on Jenkins restart,
  - if your scripts require to be executed in a particular order, you can use the DependsOn field in Custom Resource,
  - JenkinsGroovyScripts are executed after JenkinsConfigurationAsCode.

## Operator State

Operator state is kept in the custom resource status section, which is used for storing any configuration events or job statuses managed by the operator.

It helps to maintain or recover the desired state even after the operator or Jenkins restarts.
