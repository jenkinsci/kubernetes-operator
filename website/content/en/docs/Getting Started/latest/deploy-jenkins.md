---
title: "Deploy Jenkins"
linkTitle: "Deploy Jenkins"
weight: 2
date: 2021-01-25
description: >
  Deploy production ready Jenkins Operator manifest
---

Once Jenkins Operator is up and running let's deploy actual Jenkins instance.
Create manifest e.g. **`jenkins_instance.yaml`** with following data and save it on drive.

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
  namespace: default
spec:
  configurationAsCode:
    configurations: []
    secret:
      name: ""
  groovyScripts:
    configurations: []
    secret:
      name: ""
  jenkinsAPISettings:
    authorizationStrategy: createUser
  master:
    disableCSRFProtection: false
    containers:
      - name: jenkins-master
        image: jenkins/jenkins:2.277.4-lts-alpine
        imagePullPolicy: Always
        livenessProbe:
          failureThreshold: 12
          httpGet:
            path: /login
            port: http
            scheme: HTTP
          initialDelaySeconds: 100
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 5
        readinessProbe:
          failureThreshold: 10
          httpGet:
            path: /login
            port: http
            scheme: HTTP
          initialDelaySeconds: 80
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        resources:
          limits:
            cpu: 1500m
            memory: 3Gi
          requests:
            cpu: "1"
            memory: 500Mi
  seedJobs:
    - id: jenkins-operator
      targets: "cicd/jobs/*.jenkins"
      description: "Jenkins Operator repository"
      repositoryBranch: master
      repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
```

Deploy a Jenkins to Kubernetes:

```bash
kubectl create -f jenkins_instance.yaml
```
Watch the Jenkins instance being created:

```bash
kubectl get pods -w
```

Get the Jenkins credentials:

```bash
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.password}' | base64 -d
```

Connect to the Jenkins instance (minikube):

```bash
minikube service jenkins-operator-http-<cr_name> --url
```

Connect to the Jenkins instance (actual Kubernetes cluster):

```bash
kubectl port-forward jenkins-<cr_name> 8080:8080
```
Then open browser with address `http://localhost:8080`.

![jenkins](/kubernetes-operator/img/jenkins.png)
