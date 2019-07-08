# Jenkins Security

By default **jenkins-operator** performs an initial security hardening of Jenkins instance via groovy scripts to prevent any security gaps.

## Jenkins Access Control

Currently **jenkins-operator** generates a username and random password and stores them in a Kubernetes Secret.
However any other authorization mechanisms are possible and can be done via groovy scripts or configuration as code plugin.
For more information take a look at [getting-started#jenkins-customization](getting-started.md#jenkins-customisation).

Any change to Security Realm or Authorization requires that user called `jenkins-operator` must have admin rights 
because **jenkins-operator** calls Jenkins API.

## Jenkins Hardening

The list below describes all the default security setting configured by the **jenkins-operator**:
- basic settings - use `Mode.EXCLUSIVE` - Jobs must specify that they want to run on master node
- enable CSRF - Cross Site Request Forgery Protection is enabled
- disable usage stats - Jenkins usage stats submitting is disabled
- enable master access control - Slave To Master Access Control is enabled
- disable old JNLP protocols - `JNLP3-connect`, `JNLP2-connect` and `JNLP-connect` are disabled
- disable CLI - CLI access of `/cli` URL is disabled
- configure kubernetes-plugin - secure configuration for Kubernetes plugin

If you would like to dig a little bit into the code, take a look [here](../pkg/controller/jenkins/configuration/base/resources/base_configuration_configmap.go).

## Jenkins API

The **jenkins-operator** generates and configures Basic Authentication token for Jenkins go client and stores it in a Kubernetes Secret.

## Kubernetes

Kubernetes API permissions are limited by the following roles:
- [jenkins-operator role](../deploy/role.yaml)  
- [Jenkins Master role](../pkg/controller/jenkins/configuration/base/resources/rbac.go)

## Report a Security Vulnerability

If you find a vulnerability or any misconfiguration in Jenkins, please report it in the [issues](https://github.com/jenkinsci/kubernetes-operator/issues). 


