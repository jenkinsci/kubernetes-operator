---
title: "Installing the Operator"
linkTitle: "Installing the Operator"
weight: 1
date: 2021-08-20
description: >
  How to install Jenkins Operator
---

{{% pageinfo %}}
This document describes installation procedure for **Jenkins Operator**. 
All container images can be found at [virtuslab/jenkins-operator](https://hub.docker.com/r/virtuslab/jenkins-operator) Docker Hub repository.
{{% /pageinfo %}}

## Requirements
 
To run **Jenkins Operator**, you will need:

- access to a Kubernetes cluster version `1.17+`
- `kubectl` version `1.17+`


Listed below are the two ways to deploy Jenkins Operator.

## Deploy Jenkins Operator using YAML's

First, install Custom Resource Definitions used by Jenkins Operator:

```bash
kubectl apply -f
```

Then, install the Operator and other required resources:

```bash
kubectl apply -f
```

Watch **Jenkins Operator** instance being created:

```bash
kubectl get pods -w
```

Now **Jenkins Operator** should be up and running in the `default` namespace.
For deploying Jenkins, refer to [Deploy Jenkins section](/kubernetes-operator/docs/getting-started/latest/deploying-jenkins/).

## Deploy Jenkins Operator using Helm Chart

Alternatively, you can also use Helm to install the Operator (and optionally, by default, Jenkins). It requires the Helm 3+ for deployment.

Create a namespace for the operator:

```bash
$ kubectl create namespace <your-namespace>
```

To install, you need only to type these commands:

```bash
$ helm repo add jenkins https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/chart
$ helm install <name> jenkins/jenkins-operator -n <your-namespace>
```

To add custom labels and annotations, you can use `values.yaml` file or pass them into `helm install` command, e.g.:

```bash
$ helm install <name> jenkins/jenkins-operator -n <your-namespace> -f values.yaml
```

```bash
$ helm install <name> jenkins/jenkins-operator -n <your-namespace> --set jenkins.labels.LabelKey=LabelValue,jenkins.annotations.AnnotationKey=AnnotationValue
```

`values.yaml` file can be found [here](https://github.com/jenkinsci/kubernetes-operator/blob/v0.6.0/chart/jenkins-operator/values.yaml).


## Note on Operator's nightly built images
If you wish to use the newest, not yet released version of the Operator, you can use one of nightly built snapshot images, however the maintainers of this project cannot guarantee their stability.

You can find nightly built images by heading to [virtuslab/jenkins-operator](https://hub.docker.com/r/virtuslab/jenkins-operator) Docker Hub repository and looking for images with tag in the form of `{git-hash}`, {git-hash} being the hash of master branch commit that you want to use snapshot of.

## Note on restricted Jenkins controller pod volumeMounts
Current design of the Operator puts an emphasis on creating a full GitOps flow of work for Jenkins users.
One of the key points of this design is maintaining an immutable state of Jenkins. 

One of the prerequisites of this is an ephemeral Jenkins home directory. To achieve that, Operator mounts emptyDir Volume
(jenkins-home) as Jenkins home directory.
It is not possible to overwrite volumeMount and specify any other Volume for Jenkins home directory,
as attempting to do so will result in Operator error.

jenkins-home is not the only Jenkins controller pod volumeMount that is non-configurable and managed by Operator,
below is the full list of those volumeMounts:

* jenkins-home
* scripts
* init-configuration
* operator-credentials