---
title: "Separate namespaces for Jenkins and Operator"
linkTitle: "Separate namespaces for Jenkins and Operator"
weight: 7
date: 2021-08-20
description: >
    How to install Jenkins and Jenkins Operator in separate namespaces
---

## Create namespaces

You need to create two namespaces, for example we'll call them **jenkins** for Jenkins and **jenkins-operator** for Jenkins Operator.
```bash
$ kubectl create ns jenkins-operator
$ kubectl create ns jenkins
```

## Create necessary resources in Jenkins Operator namespace

Next, you need to install resources necessary for the Operator to work in the `jenkins-operator` namespace. To do that,
copy the manifest you see below to `jenkins-operator-rbac.yaml`file.

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: jenkins-operator
---
# permissions to do leader election.
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: leader-election-role
rules:
- apiGroups:
  - ""
  - coordination.k8s.io
  resources:
  - configmaps
  - leases
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: leader-election-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: leader-election-role
subjects:
- kind: ServiceAccount
  name: jenkins-operator
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jenkins-operator
rules:
- apiGroups:
  - apps
  resources:
  - daemonsets
  - deployments
  - replicasets
  - statefulsets
  verbs:
  - '*'
- apiGroups:
  - apps
  - jenkins-operator
  resources:
  - deployments/finalizers
  verbs:
  - update
- apiGroups:
  - build.openshift.io
  resources:
  - buildconfigs
  - builds
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  - services
  verbs:
  - create
  - get
  - list
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - get
  - list
  - patch
  - watch
- apiGroups:
  - ""
  resources:
  - persistentvolumeclaims
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - pods
  - pods/exec
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - pods/log
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - pods/portforward
  verbs:
  - create
- apiGroups:
  - ""
  resources:
  - serviceaccounts
  verbs:
  - create
  - get
  - list
  - update
  - watch
- apiGroups:
  - image.openshift.io
  resources:
  - imagestreams
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - jenkins.io
  resources:
  - '*'
  verbs:
  - '*'
- apiGroups:
  - jenkins.io
  resources:
  - jenkins
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - jenkins.io
  resources:
  - jenkins/finalizers
  verbs:
  - update
- apiGroups:
  - jenkins.io
  resources:
  - jenkins/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - rbac.authorization.k8s.io
  resources:
  - rolebindings
  - roles
  verbs:
  - create
  - get
  - list
  - update
  - watch
- apiGroups:
  - route.openshift.io
  resources:
  - routes
  verbs:
  - create
  - get
  - list
  - update
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jenkins-operator
subjects:
  - kind: ServiceAccount
    name: jenkins-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: jenkins-operator
```

Now install the required resources in `jenkins-operator` namespace with:
```bash
kubectl apply -n jenkins-operator -f jenkins-operator-rbac.yaml
```

There's only one thing left to install in `jenkins-operator` namespace, and that is the Operator itself. The manifest
below contains the Operator as defined in all-in-one manifest found in [Installating the Operator](/kubernetes-operator/docs/getting-started/latest/installing-the-operator/)
page, the only difference is that the one here sets `WATCH_NAMESPACE` to the `jenkins` namespace we created.

Copy its content to `jenkins-operator.yaml` file.

```bash
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jenkins-operator
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: jenkins-operator
      securityContext:
        runAsUser: 65532
      containers:
        - command:
            - /manager
          args:
            - --leader-elect
          image: virtuslab/jenkins-operator:v0.6.0
          name: jenkins-operator
          imagePullPolicy: IfNotPresent
          securityContext:
            allowPrivilegeEscalation: false
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8081
            initialDelaySeconds: 15
            periodSeconds: 20
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            limits:
              cpu: 100m
              memory: 30Mi
            requests:
              cpu: 100m
              memory: 20Mi
          env:
            - name: WATCH_NAMESPACE
              value: jenkins
      terminationGracePeriodSeconds: 10
```

Install the Operator in `jenkins-operator` namespace with:

```bash
kubectl apply -n jenkins-operator -f jenkins-operator.yaml
```

You have installed the Operator in `jenkins-operator` namespace, watching for Jenkins in `jenkins` namespace. Now
there are two things left to do: creating necessary Role and RoleBinding for the Operator in `jenkins` namespace, and
deploying actual Jenkins instance there.

## Create necessary resources in Jenkins namespace

Below you can find manifest with RBAC that need to be created in `jenkins` namespace. Copy its content to `jenkins-ns-rbac.yaml` file.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jenkins-operator
rules:
  - apiGroups:
      - apps
    resources:
      - daemonsets
      - deployments
      - replicasets
      - statefulsets
    verbs:
      - '*'
  - apiGroups:
      - apps
      - jenkins-operator
    resources:
      - deployments/finalizers
    verbs:
      - update
  - apiGroups:
      - build.openshift.io
    resources:
      - buildconfigs
      - builds
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - configmaps
      - secrets
      - services
    verbs:
      - create
      - get
      - list
      - update
      - watch
  - apiGroups:
      - ""
    resources:
      - events
    verbs:
      - create
      - get
      - list
      - patch
      - watch
  - apiGroups:
      - ""
    resources:
      - persistentvolumeclaims
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
  - apiGroups:
      - ""
    resources:
      - pods
      - pods/exec
    verbs:
      - '*'
  - apiGroups:
      - ""
    resources:
      - pods/log
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - pods/portforward
    verbs:
      - create
  - apiGroups:
      - ""
    resources:
      - serviceaccounts
    verbs:
      - create
      - get
      - list
      - update
      - watch
  - apiGroups:
      - image.openshift.io
    resources:
      - imagestreams
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - jenkins.io
    resources:
      - '*'
    verbs:
      - '*'
  - apiGroups:
      - jenkins.io
    resources:
      - jenkins
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
  - apiGroups:
      - jenkins.io
    resources:
      - jenkins/finalizers
    verbs:
      - update
  - apiGroups:
      - jenkins.io
    resources:
      - jenkins/status
    verbs:
      - get
      - patch
      - update
  - apiGroups:
      - rbac.authorization.k8s.io
    resources:
      - rolebindings
      - roles
    verbs:
      - create
      - get
      - list
      - update
      - watch
  - apiGroups:
      - route.openshift.io
    resources:
      - routes
    verbs:
      - create
      - get
      - list
      - update
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jenkins-operator
subjects:
  - kind: ServiceAccount
    name: jenkins-operator
    namespace: jenkins-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: jenkins-operator
```

Now apply it with:
```bash
kubectl apply -n jenkins -f jenkins-ns-rbac.yaml
```

The last thing to do is to deploy Jenkins. Below you can find an example Jenkins resource manifest.
It's the same as one used in [Deploying Jenkins](/kubernetes-operator/docs/getting-started/latest/deploying-jenkins/).
Copy it to `jenkins-instance.yaml`

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
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

Now you can deploy it with:

```bash
kubectl apply -n jenkins -f jenkins-instance.yaml
```

With this, you have just set up Jenkins Operator and Jenkins in separate namespaces. Now the Operator will run in
its own namespace (`jenkins-operator`), watch for CRs in `jenkins` namespace, and deploy Jenkins there.
