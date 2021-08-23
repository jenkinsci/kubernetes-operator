---
title: "Configuring Seed Jobs and Pipelines"
linkTitle: "Configuring Seed Jobs and Pipelines"
weight: 4
date: 2021-08-19
description: >
  How to configure Jenkins with Operator
---

## Configure Seed Jobs and Pipelines

Jenkins operator uses [job-dsl][job-dsl] and [kubernetes-credentials-provider][kubernetes-credentials-provider] plugins
for configuring jobs and deploy keys.

To preserve your Jenkins pipelines automate their recreation in case of instance failures, we strongly recommend using
Configuration as Code files to set up pipelines.

## Prepare job definitions and pipelines

First you have to prepare pipelines and job definition in your GitHub repository using the following structure:

```
cicd/
├── jobs
│   └── build.jenkins
└── pipelines
    └── build.jenkins
```

Jenkins will always check the configurations directly from those files in your repository, so you don’t have to update
the configuration every time you change the pipeline code itself.

### Seed Job definition
A seed job represents Jenkins job that creates one or more pipelines in Jenkins. It uses pipeline configuration files
from your GitHub cicd folder. Let’s create a job configuration file and store it at
`https://github.com/your-project-repo/cicd/jobs/build.jenkins`:

```groovy
#!/usr/bin/env groovy

pipelineJob('build-jenkins-operator') {
    displayName('Build Jenkins Operator')

    definition {
        cpsScm {
            scm {
                git {
                    remote {
                        url('https://github.com/your-account/your-repo.git')
                    }
                    branches('*/master')
                }
            }
            scriptPath('cicd/pipelines/build.jenkins')
        }
    }
}
```

* **_pipelineJob_** – name of the pipeline resource that we will create
* **_displayName_** – name of the seed job that will be displayed in Jenkins UI
* **_remote_** – here you specify the url of GitHub repository of your project
* **_branches_** – branches from which you want to access the repo
* **_scriptPath_** – the path in the above repo in which you store pipeline files from which you want to create Jenkins
  pipelines during this job run

### Pipeline definition
Now we can create the pipeline configuration file and store it at https://github.com/your-project-repo/cicd/pipelines/build.jenkins.
This file will create pod with containers to run commands on each step (stage) in your pipeline.

```groovy
#!/usr/bin/env groovy

def label = "jenkins-example-${UUID.randomUUID().toString()}"

podTemplate(label: label,
        containers: [
                containerTemplate(name: 'jnlp', image: 'jenkins/inbound-agent:alpine'),
        ],
        ) {
    node(label) {
        stage('Init') {
            timeout(time: 3, unit: 'MINUTES') {
                checkout scm
            }
        }
        stage('Dep') {
            echo 'Hello from Dep stage'
        }
        stage('Test') {
            echo 'Hello from Test stage'
        }
        stage('Build') {
            echo 'Hello from Build stage'
        }
    }
}
```

* **_label_** – the name of the pipeline
* **_podTemplate_** – a pod that will be created during this pipeline run, to execute necessary steps
* **_containers_** – containers in a pod that will be used to run necessary steps. One jnlp container is always needed. 
  All the others need to use the images of programs needed for the pipeline’s stages. Full list of possible functionalities
  can be found here. If you need to run kubectl commands you need to use container with an image that incorporates kubectl,
  because it is not provided by default.
* **_stage_** – at each stage you specify stage name, scripts, commands and container which needs to run them

## Update JenkinsSeedJob Custom Resource
When you create a seed job and pipeline files, you need to reference it and specify its details in the JenkinsSeedJob
Custom Resource. Jenkins Operator will create a default JenkinsKubernetesAgent “operator-agent” and seed job will be
processed by default agent. Jenkins will then create Jenkins jobs from seed job files from your git repository and if
you run it in the Jenkins UI, they will create the necessary pipelines.

```yaml
apiVersion: jenkins.io.com/v1beta1
kind: JenkinsSeedJob
metadata:
  name: example-jenkins-seedjob
  namespace: default
  labels:
    jenkins.io/jenkins: example
spec:
  repository:
    url: https://github.com/jenkinsci/kubernetes-operator.git
    branch: master
    targets: "cicd/jobs/*.jenkins"
```

*Note: you have to specify the Jenkins Custom Resource name in the label jenkins.io/jenkins to connect the
Seed Job with your Jenkins instance.*

If you want to use your own Kubernetes Agent for seed job, you need to add AgentRef to JenkinsSeedJob:

```yaml
apiVersion: jenkins.io/v1beta1
kind: JenkinsSeedJob
metadata:
  name: example-jenkins-seedjob
  namespace: default
  labels:
    operator-service.com/jenkins: example
spec:
  repository:
    url: https://github.com/jenkinsci/kubernetes-operator.git
    branch: master
    targets: "cicd/jobs/*.jenkins"
  agentRef:
    name: agent-name
    namespace: default
```

Jenkins Operator will then automatically discover and configure all the seed jobs. You can verify if deploy keys were
successfully configured in the Jenkins Credentials tab.


## Authentication
If your GitHub repository is private or you need to authenticate to any other applications, you have to configure SSH
or username/password authentication.

Using SSH Keys is a more secure option, while username/password method is good enough with solutions incorporating
central location redirecting users for authentication or with multistep authentication.

### Basic SSH authentication
#### Generate SSH Keys
To generate a private/public pair of keys, run ssh-keygen:

```bash
$ ssh-keygen -t rsa -b 2048 -C "johndoe@email.com"
Generating public/private rsa key pair.
```

Next, you will be asked for a file name. Use the default path:

```bash
Enter file in which to save the key (/Users/johndoe/.ssh/id_rsa):
```

Now you can optionally set a password.
```bash
Enter passphrase (empty for no passphrase):
Enter same passphrase again:
Your identification has been saved in /Users/johndoe/.ssh/id_rsa.
Your public key has been saved in /Users/johndoe/.ssh/id_rsa.pub.
The key fingerprint is:
SHA256:M0HppoJgPAhw2NYS0SVuYpyllpA7MbFfu+U0F0y8EDA johndoe@email.com
The key's randomart image is:
+---[RSA 2048]----+
|      o. .   o   |
|     o .o + +    |
|   ...+o . + .   |
|  E .ooo .. .    |
|   o  + S . .o   |
|  .  o   . o. +  |
| .    .   .o.=   |
|  . .... .o.@..  |
| .oo.o .o+=@*=   |
+----[SHA256]-----+
```

Operator needs the key in PEM format:
```bash
$ ssh-keygen -p -f /Users/johndoe/.ssh/id_rsa -m pem
Key has comment 'johndoe@email.com'
Enter new passphrase (empty for no passphrase):
Enter same passphrase again:
Your identification has been saved with the new passphrase.
```

Now copy the content of the **public** key file (the one with `.pub` in the file name) and add it to your GitHub repository.
In your project repository enter Settings > Deploy keys and click “Add deploy key”. Give it some title and paste the key
you just copied there.

![jenkins](/kubernetes-operator/img/key-deployment.png)

You should see that the key was added.

![jenkins](/kubernetes-operator/img/deployed-key.png)

#### Configure SSH authentication
First, create a Secret file with your GitHub username and generated SSH private key.

Copy the content of the **id_rsa** file (not **id_rsa.pub**) and paste it into privateKey field like this:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k8s-ssh
  labels:
    "operator-service.com/jenkinsseedjob": "example-jenkins-seedjob"
    "operator-service.com/credentials-type": "basicSSHUserPrivateKey"
stringData:
  privateKey: |
   -----BEGIN PRIVATE KEY-----
    MIIJKAIBAAKCAgEAxxDpleJjMCN5nusfW/AtBAZhx8UVVlhhhIKXvQ+dFODQIdzO
    oDXybs1zVHWOj31zqbbJnsfsVZ9Uf3p9k6xpJ3WFY9b85WasqTDN1xmSd6swD4N8
    ...   
  username: github_user_name
```

*Note: you have to specify the name of the JenkinsSeedJob Custom Resource in the labels to connect the secret with respective Seed Job.*

Second, create a Kubernetes Secret resource from this Secret config file.

```bash
kubectl-n jenkins apply -f mySecret.yaml
```

In the seedJob you added to your JenkinsSeedJob Custom Resource file you can specify basicSSHUserPrivateKey as
credentialType and add the name of the Secret, with your GitHub username and SSH key, in credentialID field’s value.

```yaml
apiVersion: jenkins.io/v1beta1
kind: JenkinsSeedJob
metadata:
  name: example-jenkins-seedjob
  namespace: default
  labels:
    jenkins.io/jenkins: example
spec:
  repository:
    url: git@github.com:your-account/your-repository.git
    branch: master
    targets: "cicd/jobs/*.jenkins"
    credentialType: basicSSHUserPrivateKey
    credentialID: k8s-ssh
```

Now we need to specify newly created credentials in your Seed Job definition file. They will be used to connect to
the specified GitHub repository. Don’t forget to also change the url for SSH:

```groovy
#!/usr/bin/env groovy

pipelineJob('build-operator-service-for-jenkins') {
    displayName('Build Operator Service for Jenkins')

    definition {
        cpsScm {
            scm {
                git {
                    remote {
                        url('git@github.com:your-account/your-repo.git')
                        credentials('k8s-sh')
                    }
                    branches('*/master')
                }
            }
            scriptPath('cicd/pipelines/build.jenkins')
        }
    }
}
```

### Username and password authentication
First, create a Secret file with your GitHub credentials.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k8s-user-pass
  labels:
    "jenkins.io/jenkinsseedjob": "example-jenkins-seedjob"
stringData:
  username: github_user_name
  password: password_or_token
```

Second, create a Kubernetes Secret resource from this file.

```bash
kubectl -n jenkins apply -f mySecret.yaml
```

In the seedJob you added to your Jenkins Custom Resource file you can specify usernamePassword as credentialType and
add the name of the Secret, with your GitHub credentials, in the credentialID field’s value.

```yaml
apiVersion: jenkins.io/v1beta1
kind: JenkinsSeedJob
metadata:
  name: example-jenkins-seedjob
  namespace: default
  labels:
    jenkins.io/jenkins: example
spec:
  repository:
    url: https://github.com/your-github-account/your-github-repository.git
    branch: master
    targets: "cicd/jobs/*.jenkins"
    credentialType: usernamePassword
    credentialID: k8s-user-pass
```

[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[kubernetes-credentials-provider]:https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/
[jenkins-using-credentials]:https://www.jenkins.io/doc/book/using/using-credentials/
[kubernetes-plugin]:https://www.jenkins.io/doc/pipeline/steps/kubernetes/
