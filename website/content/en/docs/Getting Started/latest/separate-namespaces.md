---
title: "Separate namespaces for Jenkins and Operator"
linkTitle: "Separate namespaces for Jenkins and Operator"
weight: 5
date: 2019-08-05
description: >
  How to install Jenkins and Jenkins Operator in separate namespaces
---

You need to create two namespaces, for example we'll call them **jenkins** for Jenkins and **jenkins-operator** for Jenkins Operator.
```bash
$ kubectl create ns jenkins-operator
$ kubectl create ns jenkins
```

Next, you need to install resources necessary for the Operator to work in the `jenkins-operator` namespace. To do that,
copy the manifest you see below to `jenkins-operator-rbac.yaml`file.

```yaml

```

Now install the required resources in `jenkins-operator` namespace with:
```bash
kubectl apply -n jenkins-operator -f jenkins-operator-rbac.yaml
```

There's only one thing left to install in `jenkins-operator` namespace, and that is the Operator itself. The manifest
below contains the Operator as defined in all-in-one manifest found in [Installing the Operator](/kubernetes-operator/docs/getting-started/latest/installing-the-operator/)
page, the only difference is that the one here sets `WATCH_NAMESPACE` to the `jenkins` namespace we created.

Copy its content to `jenkins-operator.yaml` file.

```yaml

```

Install the Operator in `jenkins-operator` namespace with:

```bash
kubectl apply -n jenkins-operator -f jenkins-operator.yaml
```

You have installed the Operator in `jenkins-operator` namespace, watching for Jenkins in `jenkins` namespace. Now
there are two things left to do: creating necessary Role and RoleBinding for the Operator in `jenkins` namespace, and
deploying actual Jenkins instance there.

Below you can find manifest with RBAC that need to be created in `jenkins` namespace. Copy its content to `jenkins-ns-rbac.yaml` file.

```yaml

```

Now apply it with:
```bash
kubectl apply -n jenkins -f jenkins-ns-rbac.yaml
```

The last thing to do is to deploy Jenkins. Below you can find an example Jenkins resource manifest.
It's the same as one used in [Deploying Jenkins](/kubernetes-operator/docs/getting-started/latest/deploying-jenkins/).
Copy it to `jenkins-instance.yaml`

```yaml

```

Now you can deploy it with:

```bash
kubectl apply -n jenkins -f jenkins-instance.yaml
```

With this, you have just set up Jenkins Operator and Jenkins in separate namespaces. Now the Operator will run in
its own namespace (`jenkins-operator`), watch for CRs in `jenkins` namespace, and deploy Jenkins there.
