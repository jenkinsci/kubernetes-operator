---
title: "Deploying Jenkins"
linkTitle: "Deploying Jenkins"
weight: 2
date: 2021-08-19
description: >
  Deploy production ready Jenkins manifest
---

Once Jenkins Operator is up and running let's deploy actual Jenkins instance.
Create manifest e.g. **`jenkins_instance.yaml`** with following data and save it on drive.

```yaml
apiVersion: jenkins.io/v1beta1
kind: Jenkins
metadata:
  name: example
  namespace: default
spec:
  podSpec:
    containers:
      - name: jenkins-controller
        image: jenkins/jenkins:2.277.4-lts-alpine
        imagePullPolicy: IfNotPresent
```

Deploy Jenkins to Kubernetes:

```bash
kubectl create -f jenkins_instance.yaml
```
Watch the Jenkins instance being created:

```bash
kubectl get pods -w
```

Get the Jenkins credentials:

```bash
kubectl get secret <cr_name>-credentials -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret <cr_name>-credentials -o 'jsonpath={.data.password}' | base64 -d
```

Connect to the Jenkins instance (minikube):

```bash
minikube service <cr_name>-http --url
```

Connect to the Jenkins instance (actual Kubernetes cluster):

```bash
kubectl port-forward <cr_name> 8080:8080
```
Then open browser with address `http://localhost:8080`.

![jenkins](/kubernetes-operator/img/jenkins.png)
