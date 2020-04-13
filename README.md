# Jenkins Operator

[![Version](https://img.shields.io/badge/version-v0.4.0-brightgreen.svg)](https://github.com/jenkinsci/kubernetes-operator/releases/tag/v0.4.0)
[![Build Status](https://travis-ci.org/jenkinsci/kubernetes-operator.svg?branch=master)](https://travis-ci.org/jenkinsci/kubernetes-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkinsci/kubernetes-operator "Go Report Card")](https://goreportcard.com/report/github.com/jenkinsci/kubernetes-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/virtuslab/jenkins-operator.svg)](https://hub.docker.com/r/virtuslab/jenkins-operator/tags)

Go to [**our documentation website**](https://jenkinsci.github.io/kubernetes-operator/) for more information, with examples and guides.

![logo](/assets/jenkins_gopher_wide.png)

## What's the Jenkins Operator?

Jenkins operator is a Kubernetes native operator which fully manages Jenkins on Kubernetes.
It was built with immutability and declarative configuration as code in mind.

Out of the box it provides:
- integration with Kubernetes ([Jenkins kubernetes-plugin](https://github.com/jenkinsci/kubernetes-plugin))
- pipelines as code ([Jenkins pipelines](https://jenkins.io/doc/book/pipeline/))
- extensibility via groovy scripts (similar to [Jenkins script console](https://wiki.jenkins.io/display/JENKINS/Jenkins+Script+Console)) or ([configuration as code plugin](https://github.com/jenkinsci/configuration-as-code-plugin))
- secure defaults and hardening (see [the security section](https://jenkinsci.github.io/kubernetes-operator/docs/security/) of the documentation)

## Problem statement and goals

The main reason why we decided to implement the **Jenkins Operator** is the fact that we faced a lot of problems with standard Jenkins deployment.
We want to make Jenkins more robust, suitable for dynamic and multi-tenant environments. 

Some of the problems we want to solve:
- [installing plugins with incompatible versions or security vulnerabilities](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/customization/#install-plugins)
- [better configuration as code](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/customization/)
- [security and hardening out of the box](https://jenkinsci.github.io/kubernetes-operator/docs/security/)
- [make errors more visible for end users](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/diagnostics/)
- [backup and restore for jobs history](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/configure-backup-and-restore/)
- orphaned jobs with no JNLP connection
- handle graceful shutdown properly
- proper end to end tests for Jenkins lifecycle

## Documentation

Go to [**our documentation website**](https://jenkinsci.github.io/kubernetes-operator/) for more information.

Selected content:
1. [Installation](https://jenkinsci.github.io/kubernetes-operator/docs/installation/)
2. [Getting Started](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/)
3. [How it works](https://jenkinsci.github.io/kubernetes-operator/docs/how-it-works/)
4. [Security](https://jenkinsci.github.io/kubernetes-operator/docs/security/)
5. [Developer Guide](https://jenkinsci.github.io/kubernetes-operator/docs/developer-guide/)
5. [Jenkins Custom Resource Definition scheme](https://jenkinsci.github.io/kubernetes-operator/docs/getting-started/latest/scheme/)

## Common Jenkins issues and workarounds

- Multibranch pipelines and backup issues: https://github.com/jenkinsci/kubernetes-operator/issues/104#issuecomment-554289768

## Community & Contribution

There is a dedicated channel `#jenkins-operator` on [virtuslab-oss.slack.com](https://virtuslab-oss.slack.com) ([Invite form](https://forms.gle/X3X8qA1XMirdBuEH7))

Feel free to file [issues](https://github.com/jenkinsci/kubernetes-operator/issues) or [pull requests](https://github.com/jenkinsci/kubernetes-operator/pulls).    

Before any big pull request please consult the maintainers to ensure a common direction.

## Presentations

- [Jenkins World 2019 Lisbon](assets/Jenkins_World_Lisbon_2019%20-Jenkins_Kubernetes_Operator.pdf)

## About the authors

This project was originally developed by [VirtusLab](https://virtuslab.com/) and the following [CONTRIBUTORS](https://github.com/jenkinsci/kubernetes-operator/graphs/contributors).
