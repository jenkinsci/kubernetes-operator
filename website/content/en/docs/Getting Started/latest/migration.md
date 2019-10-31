---
title: "Migration from v0.1.x"
linkTitle: "Migration from v0.1.x"
weight: 10
date: 2019-08-05
description: >
    How to migrate from v0.1.x to v0.2.x
---

### Major Changes
#### Adding the seed job agent

From version `v0.2.0` seed jobs are not run by master executors, but by a dedicated agent deployed as a Kubernetes Pod. 

We've had disabled master executors for security reasons.

#### Replacing configuration jobs with Groovy scripts

In `v0.1.x` **Jenkins Operator** user configuration application was implemented using **Jenkins** jobs 
and this mechanism was replaced since `v0.2.0` with Groovy scripts implementing the same functionality.

As a result, the **Jenkins** configuration jobs ("Configure Seed Jobs", "jenkins-operator-base-configuration", "jenkins-operator-user-configuration") are no longer visible in **Jenkins** UI.

In `v0.1.x` you can see if any of the configuration jobs failed or succeded in the **Jenkins** UI (job build logs).
Instead, you can make sure the operator is running correctly by inspecting its logs, e.g.:

```bash
$ kubectl -n logs deployment/jenkins-operator
```

#### Making User Configuration sources configurable

In `v0.1.x` **Jenkins Operator** user configuration was stored in a `ConfigMap` and a `Secret` 
named `jenkins-operator-user-configuration-<cr_name>`, and its name was hardcoded in the operator. 

Since `v0.2.0` the user configuration can be stored in a multiple `ConfigMap` and `Secret` manifests 
and has to be explicitly pointed to with `spec.configurationAsCode.configurations` and `spec.configurationAsCode.secret`
for the Configuration as Code plugin, 
and `spec.groovyScripts.configurations` and `spec.groovyScripts.secret` for the more advanced groovy scripts.

### Migration

If you want to use `v0.1.x` operator configuration with `v0.2.x` you have to modify your Jenkins Custom Resource(s) 
and add explicit references to the existing `ConfigMap` and `Secret`, e.g.:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: <cr_name>
spec:
  ...
  configurationAsCode:
    configurations: 
    - name: jenkins-operator-user-configuration-<cr_name>
    secret:
      name: jenkins-operator-user-configuration-<cr_name>
  groovyScripts:
    configurations:
    - name: jenkins-operator-user-configuration-<cr_name>
    secret:
      name: jenkins-operator-user-configuration-<cr_name>
  ...
```
