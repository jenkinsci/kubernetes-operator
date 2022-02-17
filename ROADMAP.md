# Jenkins Operator Roadmap

This document outlines the vision and technical roadmap for [jenkinsci/kubernetes-operator](https://github.com/jenkinsci/kubernetes-operator) project.

## Project Vision

With Jenkins Operator project we want to enable our community to run Jenkins in cloud-native environments. Also, support most of the public cloud providers (AWS, Azure, GCP) in terms of additional capabilities like backups, observability and cloud security.

With declarative configuration and full lifecycle management based on [Operator Framework](https://operatorframework.io/) this can become the de facto standard for running Jenkins on top of Kubernetes.

## Technical Roadmap
- Break down Jenkins Custom Resource into smaller parts, support multiple Custom Resource Definitions [#495](https://github.com/jenkinsci/kubernetes-operator/issues/495)
    - Introduce more granular schema for configuring Jenkins
    - Introduce independent reconciliation controllers
    - Refactor e2e tests to support testing individual reconciliation controllers
- Migrate Jenkins instance from Pod to Deployment [#497](https://github.com/jenkinsci/kubernetes-operator/issues/497)
    - Unblock easier integration with 3rd party systems e.g. sidecars injection
- Improve contribution process and establish governance model [#496](https://github.com/jenkinsci/kubernetes-operator/issues/496)
    - Improve CONTRIBUTING.md
    - Introduce architecture decision proposals process
    - Introduce governance model
- Engage with community more
    - After releasing Operator version with new API, gather feedback from users on where the Operator should go next
- Reference Architecture - Jenkins on Kubernetes
    - Bridge the gap between Jenkins and Kubernetes
    - https://www.jenkins.io/blog/2020/12/04/gsod-project-report/


## Have a Question?

In case of questions, feel free to create a thread on [Jenkins Operator Category](https://community.jenkins.io/c/contributing/jenkins-operator/20) of Jenkins Community Discourse or contact us directly.
