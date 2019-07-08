# Getting Started

This document describes a getting started guide for **jenkins-operator** and an additional configuration.

1. [First Steps](#first-steps)
2. [Deploy Jenkins](#deploy-jenkins)
3. [Configure Seed Jobs and Pipelines](#configure-seed-jobs-and-pipelines)
4. [Install Plugins](#install-plugins)
5. [Configure Backup & Restore](#configure-backup-and-restore)
6. [AKS](#aks)
7. [Jenkins login credentials](#jenkins-login-credentials)
8. [Override default Jenkins container command](#override-default-Jenkins-container-command)
9. [Debugging](#debugging)

## First Steps

Prepare your Kubernetes cluster and set up access.
Once you have running Kubernetes cluster you can focus on installing **jenkins-operator** according to the [Installation](installation.md) guide.

## Deploy Jenkins

Once jenkins-operator is up and running let's deploy actual Jenkins instance.
Create manifest ie. **jenkins_instance.yaml** with following data and save it on drive.

```bash
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  master:
    containers:
    - name: jenkins-master
      image: jenkins/jenkins:lts
      imagePullPolicy: Always
      livenessProbe:
        failureThreshold: 12
        httpGet:
          path: /login
          port: http
          scheme: HTTP
        initialDelaySeconds: 80
        periodSeconds: 10
        successThreshold: 1
        timeoutSeconds: 5
      readinessProbe:
        failureThreshold: 3
        httpGet:
          path: /login
          port: http
          scheme: HTTP
        initialDelaySeconds: 30
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

Deploy Jenkins to K8s:

```bash
kubectl create -f jenkins_instance.yaml
```
Watch Jenkins instance being created:

```bash
kubectl get pods -w
```

Get Jenkins credentials:

```bash
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.password}' | base64 -d
```

Connect to Jenkins (minikube):

```bash
minikube service jenkins-operator-http-<cr_name> --url
```

Connect to Jenkins (actual Kubernetes cluster):

```bash
kubectl port-forward jenkins-<cr_name> 8080:8080
```
Then open browser with address `http://localhost:8080`.
![jenkins](../assets/jenkins.png)

## Configure Seed Jobs and Pipelines

Jenkins operator uses [job-dsl][job-dsl] and [kubernetes-credentials-provider][kubernetes-credentials-provider] plugins for configuring jobs
and deploy keys.

## Prepare job definitions and pipelines

First you have to prepare pipelines and job definition in your GitHub repository using the following structure:

```
cicd/
├── jobs
│   └── build.jenkins
└── pipelines
    └── build.jenkins
```

**cicd/jobs/build.jenkins** it's a job definition:

```
#!/usr/bin/env groovy

pipelineJob('build-jenkins-operator') {
    displayName('Build jenkins-operator')

    definition {
        cpsScm {
            scm {
                git {
                    remote {
                        url('https://github.com/jenkinsci/kubernetes-operator.git')
                        credentials('jenkins-operator')
                    }
                    branches('*/master')
                }
            }
            scriptPath('cicd/pipelines/build.jenkins')
        }
    }
}
```

**cicd/jobs/build.jenkins** it's an actual Jenkins pipeline:

```
#!/usr/bin/env groovy

def label = "build-jenkins-operator-${UUID.randomUUID().toString()}"
def home = "/home/jenkins"
def workspace = "${home}/workspace/build-jenkins-operator"
def workdir = "${workspace}/src/github.com/jenkinsci/kubernetes-operator/"

podTemplate(label: label,
        containers: [
                containerTemplate(name: 'jnlp', image: 'jenkins/jnlp-slave:alpine'),
                containerTemplate(name: 'go', image: 'golang:1-alpine', command: 'cat', ttyEnabled: true),
        ],
        envVars: [
                envVar(key: 'GOPATH', value: workspace),
        ],
        ) {

    node(label) {
        dir(workdir) {
            stage('Init') {
                timeout(time: 3, unit: 'MINUTES') {
                    checkout scm
                }
                container('go') {
                    sh 'apk --no-cache --update add make git gcc libc-dev'
                }
            }

            stage('Dep') {
                container('go') {
                    sh 'make dep'
                }
            }

            stage('Test') {
                container('go') {
                    sh 'make test'
                }
            }

            stage('Build') {
                container('go') {
                    sh 'make build'
                }
            }
        }
    }
}
```

## Configure Seed Jobs

Jenkins Seed Jobs are configured using `Jenkins.spec.seedJobs` section from your custom resource manifest:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  seedJobs:
  - id: jenkins-operator
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
```

**jenkins-operator** will automatically discover and configure all seed jobs.

You can verify if deploy keys were successfully configured in Jenkins **Credentials** tab.

![jenkins](../assets/jenkins-credentials.png)

You can verify if your pipelines were successfully configured in Jenkins Seed Job console output.

![jenkins](../assets/jenkins-seed.png)

If your GitHub repository is **private** you have to configure SSH or username/password authentication.

### SSH authentication

Configure seed job like:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  seedJobs:
  - id: jenkins-operator-ssh
    credentialType: basicSSHUserPrivateKey
    credentialID: k8s-ssh
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: git@github.com:jenkinsci/kubernetes-operator.git
```

and create Kubernetes Secret(name of secret should be the same from `credentialID` field):

```
apiVersion: v1
kind: Secret
metadata:
  name: k8s-ssh
data:
  privateKey: |
    -----BEGIN RSA PRIVATE KEY-----
    MIIJKAIBAAKCAgEAxxDpleJjMCN5nusfW/AtBAZhx8UVVlhhhIKXvQ+dFODQIdzO
    oDXybs1zVHWOj31zqbbJnsfsVZ9Uf3p9k6xpJ3WFY9b85WasqTDN1xmSd6swD4N8
    ...
  username: github_user_name
```

### Username & password authentication

Configure seed job like:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  seedJobs:
  - id: jenkins-operator-user-pass
    credentialType: usernamePassword
    credentialID: k8s-user-pass
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
```

and create Kubernetes Secret(name of secret should be the same from `credentialID` field):

```
apiVersion: v1
kind: Secret
metadata:
  name: k8s-user-pass
data:
  username: github_user_name
  password: password_or_token
```

## Jenkins Customisation

Jenkins can be customized using groovy scripts or configuration as code plugin. All custom configuration is stored in
the **jenkins-operator-user-configuration-<cr_name>** ConfigMap which is automatically created by **jenkins-operator**.

**jenkins-operator** creates **jenkins-operator-user-configuration-<cr_name>** secret where user can store sensitive 
information used for custom configuration. If you have entry in secret named `PASSWORD` then you can use it in 
Configuration as Plugin as `adminAddress: "${PASSWORD}"`.

```
kubectl get secret jenkins-operator-user-configuration-<cr_name> -o yaml

kind: Secret
apiVersion: v1
type: Opaque
metadata:
  name: jenkins-operator-user-configuration-<cr_name>
  namespace: default
data:
  SECRET_JENKINS_ADMIN_ADDRESS: YXNkZgo=

```

```
kubectl get configmap jenkins-operator-user-configuration-<cr_name> -o yaml

apiVersion: v1
data:
  1-configure-theme.groovy: |2
    import jenkins.*
    import jenkins.model.*
    import hudson.*
    import hudson.model.*
    import org.jenkinsci.plugins.simpletheme.ThemeElement
    import org.jenkinsci.plugins.simpletheme.CssTextThemeElement
    import org.jenkinsci.plugins.simpletheme.CssUrlThemeElement

    Jenkins jenkins = Jenkins.getInstance()

    def decorator = Jenkins.instance.getDescriptorByType(org.codefirst.SimpleThemeDecorator.class)

    List<ThemeElement> configElements = new ArrayList<>();
    configElements.add(new CssTextThemeElement("DEFAULT"));
    configElements.add(new CssUrlThemeElement("https://cdn.rawgit.com/afonsof/jenkins-material-theme/gh-pages/dist/material-light-green.css"));
    decorator.setElements(configElements);
    decorator.save();

    jenkins.save()
  1-system-message.yaml: |2
    jenkins:
      systemMessage: "Configuration as Code integration works!!!"
      adminAddress: "${SECRET_JENKINS_ADMIN_ADDRESS}"
kind: ConfigMap
metadata:
  name: jenkins-operator-user-configuration-<cr_name>
  namespace: default
``` 

When **jenkins-operator-user-configuration-<cr_name>** ConfigMap is updated Jenkins automatically 
runs the **jenkins-operator-user-configuration** Jenkins Job which executes all scripts then
runs the **jenkins-operator-user-configuration-casc** Jenkins Job which applies Configuration as Code configuration.

## Install Plugins

Edit CR under `spec.master.plugins`:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  master:
   plugins:
   - name: simple-theme-plugin
     version: 0.5.1
```

Then **jenkins-operator** will automatically install plugins after Jenkins master pod restart.

## Configure backup and restore

Backup and restore is done by container sidecar.

### PVC

#### Create PVC

Save to file pvc.yaml:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: <pvc_name>
  namespace: <namesapce>
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
```

Run command:
```bash
$ kubectl -n <namesapce> create -f pvc.yaml
```

#### Configure Jenkins CR

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: <cr_name>
  namespace: <namespace>
spec:
  master:
    securityContext:
      runAsUser: 1000
      fsGroup: 1000
    containers:
    - name: jenkins-master
      image: jenkins/jenkins:lts
    - name: backup # container responsible for backup and restore
      env:
      - name: BACKUP_DIR
        value: /backup
      - name: JENKINS_HOME
        value: /jenkins-home
      image: virtuslab/jenkins-operator-backup-pvc:v0.0.3 # look at backup/pvc directory
      imagePullPolicy: IfNotPresent
      volumeMounts:
      - mountPath: /jenkins-home # Jenkins home volume
        name: jenkins-home
      - mountPath: /backup # backup volume
        name: backup
    volumes:
    - name: backup # PVC volume where backups will be stored
      persistentVolumeClaim:
        claimName: <pvc_name>
  backup:
    containerName: backup # container name is responsible for backup
    action:
      exec:
        command:
        - /home/user/bin/backup.sh # this command is invoked on "backup" container to make backup, for example /home/user/bin/backup.sh <backup_number>, <backup_number> is passed by operator
    interval: 30 # how often make backup in seconds
    makeBackupBeforePodDeletion: true # make backup before pod deletion
  restore:
    containerName: backup # container name is responsible for restore backup
    action:
      exec:
        command:
        - /home/user/bin/restore.sh # this command is invoked on "backup" container to make restore backup, for example /home/user/bin/restore.sh <backup_number>, <backup_number> is passed by operator
    #recoveryOnce: <backup_number> # if want to restore specific backup configure this field and then Jenkins will be restarted and desired backup will be restored
```

## AKS

Azure AKS managed Kubernetes service adds to every pod the following envs:

```yaml
- name: KUBERNETES_PORT_443_TCP_ADDR
  value:
- name: KUBERNETES_PORT
  value: tcp://
- name: KUBERNETES_PORT_443_TCP
  value: tcp://
- name: KUBERNETES_SERVICE_HOST
  value:
```

The operator is aware of it and omits these envs when checking if Jenkins pod envs have been changed. It prevents 
restart Jenkins pod over and over again.

## Jenkins login credentials

The operator automatically generate Jenkins user name and password and stores it in Kubernetes secret named 
`jenkins-operator-credentials-<cr_name>` in namespace where Jenkins CR has been deployed.

If you want change it you can override the secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: jenkins-operator-credentials-<cr-name>
  namespace: <namespace>
data:
  user: <base64-encoded-new-username>
  password: <base64-encoded-new-password>
```

If needed **jenkins-operator** will restart Jenkins master pod and then you can login with the new user and password 
credentials.

## Override default Jenkins container command

The default command for the Jenkins master container `jenkins/jenkins:lts` looks like:

```yaml
command:
- bash
- -c
- /var/jenkins/scripts/init.sh && /sbin/tini -s -- /usr/local/bin/jenkins.sh
```

The script`/var/jenkins/scripts/init.sh` is provided be the operator and configures init.groovy.d(creates Jenkins user) 
and installs plugins.
The `/sbin/tini -s -- /usr/local/bin/jenkins.sh` command runs the Jenkins master main process.

You can overwrite it in the following pattern:

```yaml
command:
- bash
- -c
- /var/jenkins/scripts/init.sh && <custom-code-here> && /sbin/tini -s -- /usr/local/bin/jenkins.sh
```

## Debugging

Turn on debug in **jenkins-operator** deployment:

```bash
sed -i 's|\(args:\).*|\1\ ["--debug"\]|' deploy/operator.yaml
kubectl apply -f deploy/operator.yaml
```

Watch Kubernetes events:

```bash
kubectl get events --sort-by='{.lastTimestamp}'
```

Verify Jenkins master logs:

```bash
kubectl logs -f jenkins-<cr_name>
```

Verify jenkins-operator logs:

```bash
kubectl logs deployment/jenkins-operator
```

## Troubleshooting

Delete Jenkins master pod and wait for the new one to come up:

```bash
kubectl delete pod jenkins-<cr_name>
```

[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[kubernetes-credentials-provider]:https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/
