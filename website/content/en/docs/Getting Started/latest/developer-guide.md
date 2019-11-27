---
title: "Developer Guide"
linkTitle: "Developer Guide"
weight: 60
date: 2019-08-05
description: >
  Jenkins Operator for developers
---

{{% pageinfo %}}
This document explains how to setup your development environment.
{{% /pageinfo %}}

## Prerequisites

- [operator_sdk][operator_sdk] version v0.8.1
- [git][git_tool]
- [go][go_tool] version v1.12+
- [goimports, golint, checkmake and staticcheck][install_dev_tools]
- [minikube][minikube] version v1.1.0+ (preferred Hypervisor - [virtualbox][virtualbox])
- [docker][docker_tool] version 17.03+

## Clone repository and download dependencies

```bash
mkdir -p $GOPATH/src/github.com/jenkinsci
cd $GOPATH/src/github.com/jenkinsci/
git clone git@github.com:jenkinsci/kubernetes-operator.git
cd kubernetes-operator
make go-dependencies
```

## Build and run with a minikube

Build and run **Jenkins Operator** locally:

```bash
make build minikube-run EXTRA_ARGS='--minikube --local'
```

Once minikube and **Jenkins Operator** are up and running, apply Jenkins custom resource:

```bash
kubectl apply -f deploy/crds/jenkins_v1alpha2_jenkins_cr.yaml
kubectl get jenkins -o yaml
kubectl get po
```

## Build and run with a remote Kubernetes cluster

You can also run the controller locally and make it listen to a remote Kubernetes server.

```bash
make run NAMESPACE=default KUBECTL_CONTEXT=remote-k8s EXTRA_ARGS='--kubeconfig ~/.kube/config'
```

Once minikube and **Jenkins Operator** are up and running, apply Jenkins custom resource:

```bash
kubectl --context remote-k8s --namespace default apply -f deploy/crds/jenkins_v1alpha2_jenkins_cr.yaml
kubectl --context remote-k8s --namespace default get jenkins -o yaml
kubectl --context remote-k8s --namespace default get po
```

## Testing

Run unit tests:

```bash
make test
```

### Running E2E tests on Linux

Run e2e tests with minikube:

```bash
make minikube-start
eval $(minikube docker-env)
make build e2e
```

Run the specific e2e test:

```bash
make build e2e E2E_TEST_SELECTOR='^TestConfiguration$'
```

### Running E2E tests on macOS

At first, you need to start minikube:
```bash
$ make minikube-start
$ eval $(minikube docker-env) 
```

Build a Docker image inside the provided Linux container by:
```bash
$ make indocker
```

Build **Jenkins Operator** inside a container using:


```bash
$ make build
```

Then exit the container and run:
```
make e2e
```

## Tips & Tricks

### Building docker image on minikube (for e2e tests)

To be able to work with the docker daemon on `minikube` machine run the following command before building an image:

```bash
eval $(minikube docker-env)
```

### When `pkg/apis/jenkinsio/*/jenkins_types.go` has changed

Run:

```bash
make deepcopy-gen
```

### Getting the Jenkins URL and basic credentials

```bash
minikube service jenkins-operator-http-<cr_name> --url
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.password}' | base64 -d
```

# Tools

## Configure environment variables

```bash
export GOPATH=/home/go # example value
export GOROOT=/usr/lib/go-1.12 # example value
export PATH=$GOPATH/bin:$PATH
```

## goimports

```
go get golang.org/x/tools/cmd/goimports
cd $GOPATH/src/golang.org/x/tools/cmd/goimports
go build
go install
```

## golint

```
go get -u golang.org/x/lint/golint
cd  $GOPATH/src/golang.org/x/lint/golint
go build
go install
```

## checkmake
```
go get github.com/mrtazz/checkmake
cd $GOPATH/src/github.com/mrtazz/checkmake
go build
go install
```

## staticcheck

```
mkdir -p $GOPATH/src/github.com/dominikh/
cd $GOPATH/src/github.com/dominikh/
git clone https://github.com/dominikh/go-tools.git
cd  $GOPATH/src/github.com/dominikh/go-tools/staticcheck
go build
go install
```


[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[operator_sdk]:https://github.com/operator-framework/operator-sdk
[fork_guide]:https://help.github.com/articles/fork-a-repo/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube]:https://kubernetes.io/docs/tasks/tools/install-minikube/
[virtualbox]:https://www.virtualbox.org/wiki/Downloads
[jenkins-operator]:../README.md
[install_dev_tools]:install_dev_tools.md
