---
title: "Security"
linkTitle: "Security"
weight: 8
date: 2019-08-05
description: >
  Jenkins security and hardening out of the box
---

By default **Jenkins Operator** performs an initial security hardening of Jenkins instance 
via groovy scripts to prevent any security gaps.

## Jenkins Access Control

Currently **Jenkins Operator** generates a username and random password and stores them in a Kubernetes Secret.
However any other authorization mechanisms are possible and can be done via groovy scripts or configuration as code plugin.
For more information take a look at [getting-started#jenkins-customization](/kubernetes-operator/docs/getting-started/latest/customization/).

Any change to Security Realm or Authorization requires that user called `jenkins-operator` must have admin rights 
because **Jenkins Operator** calls Jenkins API.

## Jenkins Hardening

The list below describes all the default security setting configured by the **Jenkins Operator**:

- basic settings - use `Mode.EXCLUSIVE` - Jobs must specify that they want to run on master node
- enable CSRF - Cross Site Request Forgery Protection is enabled
- disable usage stats - Jenkins usage stats submitting is disabled
- enable master access control - Slave to Master Access Control is enabled
- disable old JNLP protocols - `JNLP3-connect`, `JNLP2-connect` and `JNLP-connect` are disabled
- disable CLI - CLI access of `/cli` URL is disabled
- configure kubernetes-plugin - secure configuration for Kubernetes plugin

If you would like to dig a little bit into the code, take a look [here](https://github.com/jenkinsci/kubernetes-operator/blob/master/pkg/configuration/base/resources/base_configuration_configmap.go).

## Jenkins API

The **Jenkins Operator** generates and configures Basic Authentication token for Jenkins Go client 
and stores it in a Kubernetes Secret.

## Kubernetes

Kubernetes API permissions are limited by the following roles:

- [jenkins-operator role](https://github.com/jenkinsci/kubernetes-operator/blob/v0.5.0/deploy/role.yaml)  
- [Jenkins Master role](https://github.com/jenkinsci/kubernetes-operator/blob/v0.5.0/pkg/configuration/base/resources/rbac.go)

Since **Jenkins Operator** must be able to grant permission for its' deployed Jenkins masters 
to spawn pods (the `Jenkins Master role` above), 
the operator itself requires permission to create RBAC resources (the `jenkins-operator role` above). 

Deployed this way, any subject which may create a Pod (including a Jenkins job) may 
assume the `jenkins-operator` role by using its' ServiceAccount, create RBAC rules, and thus escape its granted permissions. 
Any namespace to which the `jenkins-operator` is deployed must be considered to implicitly grant all 
possible permissions to any subject which can create a Pod in that namespace.

To mitigate this issue **Jenkins Operator** should be deployed in one namespace and the Jenkins CR should be created in separate namespace. 
To achieve it change watch namespace in [https://github.com/jenkinsci/kubernetes-operator/blob/master/deploy/operator.yaml#L25](https://github.com/jenkinsci/kubernetes-operator/blob/master/deploy/operator.yaml#L25)

## Setup Jenkins Operator and Jenkins in separate namespaces

You need to create two namespaces, for example we'll call them **jenkins** for Jenkins and **Jenkins Operator** for Jenkins Operator.
```bash
$ kubectl create ns jenkins-operator
$ kubectl create ns jenkins
```

Next, apply the RBAC manifests for **Jenkins Operator** namespace
```bash
$ kubectl -n jenkins-operator apply -f deploy/service_account.yaml
$ kubectl -n jenkins-operator apply -f deploy/role_binding.yaml
```

Create file `role_binding_jenkins.yaml` in `deploy` folder:
```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: jenkins-operator
  namespace: jenkins
subjects:
- kind: ServiceAccount
  name: jenkins-operator
  namespace: jenkins-operator
roleRef:
  kind: Role
  name: jenkins-operator
  apiGroup: rbac.authorization.k8s.io
```

Then, apply RBAC rules for **jenkins** namespace
```bash
$ kubectl -n jenkins apply -f deploy/role.yaml
$ kubectl -n jenkins apply -f role_binding_jenkins.yaml
```

Finally, you must create operator pod by:
```bash
$ kubectl -n jenkins -n jenkins-operator apply -f deploy/operator.yaml
```


## Report a Security Vulnerability

If you find a vulnerability or any misconfiguration in Jenkins, please report it in the [issues](https://github.com/jenkinsci/kubernetes-operator/issues).
