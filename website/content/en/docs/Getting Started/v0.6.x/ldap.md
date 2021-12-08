---
title: "LDAP"
linkTitle: "LDAP"
weight: 9
date: 2021-12-08
description: >
    Additional configuration for LDAP
---

Configuring LDAP is not supported out of the box, but can be achieved through
plugins and some well tuned configurations.

The plugin we will use is: <https://plugins.jenkins.io/ldap/>

> Note: This is an example of how LDAP authentication can be achieved. The LDAP
> plugin is from a third-party, and there may be other alternatives that suits
> your use case better. Use this guide with a grain of salt.

## Requirements

- LDAP server accessible from the Kubernetes cluster where your Jenkins
  instance will live.

- Credentials to a manager account in your AD. Jenkins Operator will use
  this account to authenticate with Jenkins for health checks, seed jobs, etc.

## Steps

In your Jenkins configuration, add the following plugin:

```yaml
plugins:
    # Check https://plugins.jenkins.io/ldap/ to find the latest version.
  - name: ldap
    version: "2.7"
```

Easiest step is to then start up Jenkins then navigate to your instance's
"Configure Global Security" page and configure it accordingly.

`http://jenkins.example.com/configureSecurity/`

Once it's set up and tested, you can navigate to your JCasC page and export
the LDAP settings.

`https://jenkins.example.com/configuration-as-code/`

Feed the relevant new settings into your Kubernetes ConfigMap for your JCasC
settings.

Here's a snippet of the LDAP-related configurations:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jenkins-casc
data:
  ldap.yaml: |
    jenkins:
      securityRealm:
        ldap:
          configurations:
            - displayNameAttributeName: "name"
              groupSearchBase: "OU=Groups,OU=MyCompany"
              groupSearchFilter: "(& (cn={0}) (objectclass=group) )"
              inhibitInferRootDN: false
              managerDN: "CN=Jenkins Admin,OU=UsersSystem,OU=UsersOther,OU=MyCompany,DC=mycompany,DC=local"
              managerPasswordSecret: "${LDAP_MANAGER_PASSWORD}"
              rootDN: "DC=mycompany,DC=local"
              server: "MyCompany.local"
              userSearch: "SamAccountName={0}"
              userSearchBase: "OU=MyCompany"
          disableMailAddressResolver: false
          disableRolePrefixing: true
          groupIdStrategy: "caseInsensitive"
          userIdStrategy: "caseInsensitive"
```

> Note the use of `${LDAP_MANAGER_PASSWORD}` above. You can reference
> Kubernetes secrets in your JCasC ConfigMaps by adding the following to your
> Jenkins object:
>
> ```yaml
> kind: Jenkins
> spec:
>   configurationAsCode:
>     configurations:
>       - name: jenkins-casc
>     secret:
>       # This here
>       name: jenkins-casc-secrets
> ```
>
> ```yaml
> apiVersion: v1
> kind: Secret
> metadata:
>   name: jenkins-cred-conf-secrets
> stringData:
>   LDAP_MANAGER_PASSWORD: <password-for-manager-created-in-ldap>
> ```
>
> Schema reference: [v1alpha2.ConfigurationAsCode](./schema/#github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2.ConfigurationAsCode)

Finally you must configure the Jenkins operator to use the manager's
credentials from the AD.

This is because this procedure will disable Jenkins' own user database, and the
Jenkins operator still needs to be able to talk to Jenkins in an authorized
manner.

Create the following Kubernetes secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: jenkins-operator-credentials-<jenkins-cr-name>
  namespace: <jenkins-cr-namespace>
stringData:
  user: <username-for-manager-created-in-ldap>
  password: <password-for-manager-created-in-ldap>
```

> Note: Values in stringData do not need to be base64 encoded. They are
> encoded by Kubernetes when the manifest is applied.

