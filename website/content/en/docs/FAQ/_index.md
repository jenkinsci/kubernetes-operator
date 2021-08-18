---
title: "FAQ"
linkTitle: "FAQ"
date: 2021-07-01
weight: 6
description: >
    Frequently Asked Questions about running Jenkins Operator
---

This document answers the most frequently asked questions.

### My Jenkins pod keeps restarting with ‘missing-plugins’ errors.
Jenkins can lose compatibility with its plugins or their dependencies.
If you want to reduce the probability of it happening, don’t use ‘latest’ Jenkins image tag.
Use set version of Jenkins image and declare plugins and all their dependencies in the Jenkins
Custom Resource under ‘plugins’. If you are not sure which plugins to pin, you can check the logs
from the ‘initial-config’ initcontainer or ‘jenkins-master’.

### My job fails saying I don't have necessary permissions.
You can always add a custom Role for your Jenkins with the permissions you need and reference it in the
Jenkins Custom Resource under 'spec.roles'. The Operator will create a RoleBinding for it. Be careful.
Operator may also not have these permissions. As a quick temporary workaround, you can manually bind this
role to the Operator service account.

### How can I change JENKINS_HOME from volume to Persistent Volume?
In order to provide smooth extension, scalability and errorless backups, Jenkins needs to stay ephemeral.
There is no way to change volume for JENKINS_HOME. All the configurations should be volatile in Jenkins
and kept in a VCS.
