---
title: "Security"
linkTitle: "Security"
weight: 3
date: 2026-10-31
description: >
    Jenkins security and hardening out of the box
---

By default **Jenkins Operator** applies initial hardening settings to Jenkins via Groovy scripts.

## Jenkins Access Control

Currently **Jenkins Operator** generates a username and random password and stores them in a Kubernetes Secret.
However any other authorization mechanisms are possible and can be done via groovy scripts or configuration as code plugin.
For more information take a look at the section on [customizing Jenkins](/kubernetes-operator/docs/getting-started/latest/customizing-jenkins/).

Any change to Security Realm or Authorization requires that user called `jenkins-operator` must have admin rights
because **Jenkins Operator** calls Jenkins API.

## SSO and external security realms

The operator needs non-interactive access to the Jenkins API even when people sign in
through SSO. A browser redirect, MFA prompt, or an identity-provider session cannot
replace the operator's API credentials. Keep a dedicated robot account with Jenkins
administrative permissions that can authenticate through your chosen security realm.
This Jenkins account is separate from a Kubernetes ServiceAccount.

For the default `createUser` authorization strategy, credentials are stored in the
Secret `jenkins-operator-credentials-<jenkins-cr-name>` in the Jenkins resource's
namespace. The operator uses its `user` and `password` to generate an API token,
then uses that token for API calls. It regenerates the token after Jenkins pod
replacement, so a manually supplied token alone is not a durable substitute for
working username/password authentication.

Before switching realms, ensure the robot account exists in the target realm and
retains administrative access. Keep the credentials Secret consistent with that
account. See the [LDAP example](/kubernetes-operator/docs/getting-started/latest/ldap/)
for external-directory credentials. SSO plugins differ in their support for local
accounts and API authentication; verify the selected plugin and configuration in a
development installation, including after restarting Jenkins.

If reconciliation fails after enabling SSO, check Jenkins and operator logs for
HTTP 401/403 responses or redirects to the identity provider. Confirm the account's
permissions, realm and credentials before changing configuration. Do not include
Secret contents or tokens in issue reports.

## Jenkins Hardening

The list below describes all the default security setting configured by the **Jenkins Operator**:

- basic settings - use `Mode.EXCLUSIVE` - Jobs must specify that they want to run on master node
- enable CSRF - Cross Site Request Forgery Protection is enabled
- disable usage stats - Jenkins usage stats submitting is disabled
- enable master access control - Slave to Master Access Control is enabled
- disable CLI - CLI access of `/cli` URL is disabled
- configure kubernetes-plugin - secure configuration for Kubernetes plugin

If you would like to dig a little bit into the code, take a look [here][base-configuration].

## Agent protocols

Since 0.9.0, the operator no longer explicitly overrides Jenkins agent protocol settings.
Review the protocols enabled by Jenkins and your own configuration when upgrading.

## Plugin security warnings

Plugin security validation is optional. The Helm setting `webhook.enabled=true` enables the
operator's validating webhook; `jenkins.validateSecurityWarnings=true` enables validation for
the chart-created Jenkins resource. For a separately managed resource, use
`spec.validateSecurityWarnings: true`.

The validator checks configured base and additional plugin versions against warning patterns from
`https://updates.jenkins.io/update-center.actual.json`. Matching versions cause admission to be rejected.
If the metadata has not yet been fetched, validation also rejects the resource. The operator must be
able to reach the metadata endpoint. This check is not a scan of every plugin installed dynamically in Jenkins.

See the [webhook installation instructions](/kubernetes-operator/docs/getting-started/latest/installing-the-operator/#validating-webhook).

## Jenkins API

The **Jenkins Operator** generates and configures Basic Authentication token for Jenkins Go client
and stores it in a Kubernetes Secret.

## Kubernetes

Kubernetes API permissions are limited by the following roles:

- [jenkins-operator role][jenkins-operator-role]
- [Jenkins Controller (Master) role][jenkins-controller-role]

Since **Jenkins Operator** must be able to grant permission for its deployed Jenkins masters
to spawn pods (the `Jenkins Master role` above),
the operator itself requires permission to create RBAC resources (the `jenkins-operator role` above).

Deployed this way, any subject which may create a Pod (including a Jenkins job) may
assume the `jenkins-operator` role by using its' ServiceAccount, create RBAC rules, and thus escape its granted permissions.
Any namespace to which the `jenkins-operator` is deployed must be considered to implicitly grant all
possible permissions to any subject which can create a Pod in that namespace.

To mitigate this issue, **Jenkins Operator** should be deployed in one namespace, and the Jenkins CR should be created in
a separate namespace. For instructions on how to deploy Jenkins Operator and Jenkins in separate namespaces, head over
to the [Separate namespaces](/kubernetes-operator/docs/getting-started/latest/separate-namespaces) section of Getting Started
guide.


## Report a Security Vulnerability

If you find a vulnerability or any misconfiguration in Jenkins, please report it in the [issues](https://github.com/jenkinsci/kubernetes-operator/issues).

[jenkins-operator-role]:https://github.com/jenkinsci/kubernetes-operator/blob/v0.9.0/deploy/all-in-one-v1alpha2.yaml
[jenkins-controller-role]:https://github.com/jenkinsci/kubernetes-operator/blob/v0.9.0/pkg/configuration/base/resources/rbac.go
[base-configuration]:https://github.com/jenkinsci/kubernetes-operator/blob/v0.9.0/pkg/configuration/base/resources/base_configuration_configmap.go
[issues]:https://github.com/jenkinsci/kubernetes-operator/issues
